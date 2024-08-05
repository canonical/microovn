package node

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/canonical/microcluster/cluster"
	"github.com/canonical/microcluster/state"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/snap"
)

type SrvName string

const (
	SrvChassis SrvName = "chassis"
	SrvCentral SrvName = "central"
	SrvSwitch  SrvName = "switch"
)

var ServiceNames = []SrvName{SrvChassis, SrvCentral, SrvSwitch}

func CheckValidService(service string) bool {
	return slices.Contains(ServiceNames, SrvName(service))
}

func DisableService(s *state.State, service string) error {
	exists, err := HasServiceActive(s, service)

	if err != nil {
		return err
	}
	if !exists {
		return errors.New("This service is not enabled")
	}

	if SrvName(service) == SrvCentral {
		err = snap.Stop("ovn-ovsdb-server-nb", true)
		if err != nil {
			return err
		}
		err = snap.Stop("ovn-ovsdb-server-sb", true)
		if err != nil {
			return err
		}
		err = snap.Stop("ovn-northd", true)
	} else {
		err = snap.Stop(service, true)
	}

	if err != nil {
		return fmt.Errorf("Snapctl error, likely due to service not existing:\n %w", err)
	}

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		err := database.DeleteService(ctx, tx, s.Name(), service)
		return err
	})

	return err

}

func EnableService(s *state.State, service string) error {
	exists, err := HasServiceActive(s, service)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("This Service is already enabled")
	}

	if !CheckValidService(service) {
		return errors.New("Service does not exist")
	}
	err = snap.Start(service, true)
	if err != nil {
		return fmt.Errorf("Snapctl error, likely due to service not existing:\n%w", err)
	}

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		_, err := database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: service})
		return err
	})

	return err

}

func ListServices(s *state.State) (types.Services, error) {
	services := types.Services{}

	// Get the services from the database.
	err := s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		records, err := database.GetServices(ctx, tx)
		if err != nil {
			return fmt.Errorf("Failed to fetch service: %w", err)
		}

		for _, service := range records {
			services = append(services, types.Service{
				Location: service.Member,
				Service:  service.Service,
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
func HasServiceActive(s *state.State, serviceName string) (bool, error) {
	serviceActive := false

	err := s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
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
func FindService(s *state.State, service string) ([]cluster.InternalClusterMember, error) {
	var membersWithService []cluster.InternalClusterMember

	err := s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		clusterMembers, err := cluster.GetInternalClusterMembers(ctx, tx)
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
