// Package node provides functions operating on nodes in the cluster.
package node

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/cluster"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/frr/bgp"
	"github.com/canonical/microovn/microovn/ovn/certificates"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/paths"
	"github.com/canonical/microovn/microovn/snap"
)

// DisableService - stop snap service(s) (runtime state) and remove it from the
// database (desired state).
//
// NOTE: this function does not update the environment file,
// if central is disabled then the environment files for the other nodes will be
// incorrect, please call with a method of updating the clusters env files afterwards
func DisableService(ctx context.Context, s state.State, service types.SrvName) error {
	exists, err := HasServiceActive(ctx, s, service)

	if err != nil {
		return err
	}
	if !exists {
		return errors.New("This service is not enabled")
	}

	// If going to disable central, check if possible, this is done before the
	// other check if central because we need to do a database transaction,
	// and if this check is moved into the later "if central" then we
	// will need to handle the database check on each branch of the if
	if service == types.SrvCentral {
		centrals, err := FindService(ctx, s, service)
		if err != nil {
			return err
		}
		if len(centrals) == 1 {
			return errors.New("You cannot delete the final enabled central service")
		}
	}

	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		err := database.DeleteService(ctx, tx, s.Name(), service)
		return err
	})
	if err != nil {
		return err
	}

	if service == types.SrvCentral {
		err = leaveCentral(ctx, s)
	} else if service == types.SrvChassis {
		err = leaveChassis(ctx, s)
	} else if service == types.SrvBgp {
		err = bgp.DisableService()
		if err != nil {
			return err
		}
	} else {
		err = snap.Stop(service, true)
		if err != nil {
			err = fmt.Errorf("Snapctl error, likely due to service not existing:\n %w", err)
		}
	}

	return err
}

// EnableService - start snap service(s) (runtime state) and add it to the
// database (desired state).
//
// NOTE: this function does not update the environment file,
// if central is enabled then the environment files for the other nodes will be
// incorrect, please call with a method of updating the clusters env files afterwards
func EnableService(ctx context.Context, s state.State, service types.SrvName, extraConfig *types.ExtraServiceConfig) error {
	exists, err := HasServiceActive(ctx, s, service)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("This Service is already enabled")
	}

	if !types.CheckValidService(service) {
		return errors.New("Service does not exist")
	}

	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		_, err := database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: service})
		return err
	})
	if err != nil {
		return err
	}

	if service == types.SrvCentral {
		err = joinCentral(ctx, s)
	} else if service == types.SrvChassis {
		err = joinChassis(ctx, s)
	} else if service == types.SrvBgp {
		err = bgp.EnableService(ctx, s, extraConfig.BgpConfig)
	} else {
		err = snap.Start(service, true)
		if err != nil {
			err = fmt.Errorf("Snapctl error, likely due to service not existing:\n%w", err)
		}
	}

	return err
}

// ListServices - List services in database (desired state).
func ListServices(ctx context.Context, s state.State) (types.Services, error) {
	services := types.Services{}

	// Get the services from the database.
	err := s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		records, err := database.GetServices(ctx, tx)
		if err != nil {
			return fmt.Errorf("Failed to fetch service: %w", err)
		}

		for _, service := range records {
			services = append(services, types.Service{
				Location: service.Member,
				Service:  types.SrvName(service.Service),
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return services, nil
}

// HasServiceActive function accepts service names (like "central" or "switch") and returns true/false based
// on whether the selected service is running on this node.
func HasServiceActive(ctx context.Context, s state.State, serviceName types.SrvName) (bool, error) {
	serviceActive := false

	err := s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		// Get list of all active local services.
		name := s.Name()
		services, err := database.GetServices(ctx, tx, database.ServiceFilter{Member: &name})
		if err != nil {
			return err
		}

		// Check if the specified service is among active local services.
		for _, srv := range services {
			if srv.Service == serviceName {
				serviceActive = true
			}
		}

		return nil
	})

	return serviceActive, err
}

// FindService returns list of cluster members that have the specified service enabled.
func FindService(ctx context.Context, s state.State, service types.SrvName) ([]cluster.CoreClusterMember, error) {
	var membersWithService []cluster.CoreClusterMember

	err := s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		clusterMembers, err := cluster.GetCoreClusterMembers(ctx, tx)
		if err != nil {
			return err
		}

		for _, member := range clusterMembers {
			memberHasService, err := database.ServiceExists(ctx, tx, member.Name, service)
			if err != nil {
				return err
			}

			if memberHasService {
				membersWithService = append(membersWithService, member)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find nodes with service '%s': '%s'", service, err)
	}

	return membersWithService, nil
}

// ServiceWarnings - checks the desired state and aims to find out if there are
// any problems with it, such as an inefficent or error prone number of nodes.
// This function returns a set of warnings to be handled
func ServiceWarnings(ctx context.Context, s state.State) (types.WarningSet, error) {
	output := types.WarningSet{}
	centrals, err := FindService(ctx, s, types.SrvCentral)
	if err != nil {
		return output, err
	}
	if (len(centrals) % 2) == 0 {
		output.EvenCentral = true
	}
	if len(centrals) < 3 {
		output.FewCentral = true
	}
	return output, nil
}

// joinCentral safely starts the central services child services while also
// generating certificates to ensure secure connection with other central
// nodes in the database
func joinCentral(ctx context.Context, s state.State) error {
	// Generate certificate for OVN Central services
	err := certificates.GenerateNewServiceCertificate(ctx, s, "ovnnb", certificates.CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovnnb service")
	}
	err = certificates.GenerateNewServiceCertificate(ctx, s, "ovnsb", certificates.CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovnsb service")
	}
	err = certificates.GenerateNewServiceCertificate(ctx, s, "ovn-northd", certificates.CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovn-northd service")
	}

	err = snap.Start("ovn-ovsdb-server-nb", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN NB: %w", err)
	}

	err = snap.Start("ovn-ovsdb-server-sb", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN SB: %w", err)
	}

	err = snap.Start("ovn-northd", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN northd: %w", err)
	}
	return nil
}

// leaveCentral safely stops the central service's child services, and leaves
// the central database cluster safely.
func leaveCentral(ctx context.Context, s state.State) error {
	// Leave SB and NB clusters
	logger.Info("Leaving OVN Northbound cluster")
	_, err := ovnCmd.AppCtl(ctx, s, paths.OvnNBControlSock(), "cluster/leave", "OVN_Northbound")
	if err != nil {
		logger.Warnf("Failed to leave OVN Northbound cluster: %s", err)
	}

	logger.Info("Leaving OVN Southbound cluster")
	_, err = ovnCmd.AppCtl(ctx, s, paths.OvnSBControlSock(), "cluster/leave", "OVN_Southbound")
	if err != nil {
		logger.Warnf("Failed to leave OVN Southbound cluster: %s", err)
	}

	// Wait for NB and SB cluster members to complete departure process
	nbDatabase, err := ovnCmd.NewOvsdbSpec(ovnCmd.OvsdbTypeNBLocal)
	if err == nil {
		err = ovnCmd.WaitForDBState(ctx, s, nbDatabase, ovnCmd.OvsdbRemoved, ovnCmd.DefaultDBConnectWait)
		if err != nil {
			logger.Warnf("Failed to wait for NB cluster departure: %s", err)
		}
	} else {
		logger.Warnf("Failed to get NB database specification: %s", err)
	}

	sbDatabase, err := ovnCmd.NewOvsdbSpec(ovnCmd.OvsdbTypeSBLocal)
	if err == nil {
		err = ovnCmd.WaitForDBState(ctx, s, sbDatabase, ovnCmd.OvsdbRemoved, ovnCmd.DefaultDBConnectWait)
		if err != nil {
			logger.Warnf("Failed to wait for SB cluster departure: %s", err)
		}
	} else {
		logger.Warnf("Failed to get SB database specification: %s", err)
	}

	err = os.Rename(paths.CentralDBNBPath(), paths.CentralDBNBBackupPath())
	if err != nil {
		logger.Warnf("Failed to move Northbound database to backup: %s", err)
	}

	err = os.Rename(paths.CentralDBSBPath(), paths.CentralDBSBBackupPath())
	if err != nil {
		logger.Warnf("Failed to move Southbound database to backup: %s", err)
	}

	err = snap.Stop("ovn-northd", true)
	if err != nil {
		logger.Warnf("Failed to stop OVN northd service: %s", err)
	}

	err = snap.Stop("ovn-ovsdb-server-nb", true)
	if err != nil {
		logger.Warnf("Failed to stop OVN NB service: %s", err)
	}

	err = snap.Stop("ovn-ovsdb-server-sb", true)
	if err != nil {
		logger.Warnf("Failed to stop OVN SB service: %s", err)
	}
	return nil
}

func leaveChassis(ctx context.Context, s state.State) error {
	chassisName := s.Name()

	// Gracefully exit OVN controller causing chassis to be automatically removed.
	logger.Infof("Stopping OVN Controller and removing Chassis '%s' from OVN SB database.", chassisName)
	_, err := ovnCmd.ControllerCtl(ctx, s, "exit")
	if err != nil {
		logger.Warnf("Failed to gracefully stop OVN Controller: %s", err)
	}

	err = snap.Stop(types.SrvChassis, true)
	if err != nil {
		logger.Warnf("Failed to stop Chassis service: %s", err)
	}
	return nil
}

func joinChassis(ctx context.Context, s state.State) error {
	// Generate certificate for OVN chassis (controller)
	err := certificates.GenerateNewServiceCertificate(ctx, s, "ovn-controller", certificates.CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovn-controller service")
	}
	err = snap.Start("chassis", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN chassis: %w", err)
	}
	return nil
}

// DisableAllServices is a function to disable alot of services
func DisableAllServices(ctx context.Context, s state.State) error {
	for _, service := range types.ServiceNames {
		err := DisableService(ctx, s, service)
		if err != nil {
			logger.Warnf("%s", err)
		}
	}
	return nil
}
