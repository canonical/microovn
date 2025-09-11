package ovn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/node"
	"github.com/canonical/microovn/microovn/ovn/certificates"
	ovnCluster "github.com/canonical/microovn/microovn/ovn/cluster"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/environment"
)

// Join will join an existing OVN deployment.
func Join(ctx context.Context, s state.State, initConfig map[string]string) error {
	// Make sure we don't have any other hooks firing.
	muHook.Lock()
	defer muHook.Unlock()

	// Create our storage.
	err := environment.CreatePaths()
	if err != nil {
		return err
	}

	// Query existing core services.
	srvCentral := 0
	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		// Central.
		name := "central"
		services, err := database.GetServices(ctx, tx, database.ServiceFilter{Service: &name})
		if err != nil {
			return err
		}

		srvCentral = len(services)

		return nil
	})
	if err != nil {
		return err
	}

	// The default behavior on join is to always enable chassis and switch, but enable
	// central only if:
	//   * external OVN central wasn't configured
	//   * or if there are less than 3 MicroOVN nodes with 'central' service enabled
	externalOvnCentral, err := environment.IsExternalCentralConfigured(ctx, s)
	if err != nil {
		return err
	}
	enableServices := requestedServices{
		Central: !externalOvnCentral && srvCentral < 3,
		Chassis: true,
		Switch:  true,
	}

	// Parse custom bootstrap options from initConfig
	ovnEncapIP := s.Address().Hostname()
	for k, v := range initConfig {
		// Configure OVS to either use a custom encapsulation IP for the geneve tunel
		// or the hostname of the node.
		if k == "ovn-encap-ip" {
			ovnEncapIP = v
			continue
		}

		// Get requested services
		if k == "ovn-services" {
			if v != "auto" {
				enableServices, err = newRequestedServices(v)
				if err != nil {
					return fmt.Errorf("failed to parse requested services: %w", err)
				}
			}
		}
	}

	// Generate the configuration.
	err = environment.GenerateEnvironment(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to generate the daemon configuration: %w", err)
	}

	// Generate client certificate for managing OVN Central services
	// Note that we intentially use a sever type certificate here due to
	// all OVS-based programs ability to specify active or passive (listen)
	// connection types.
	err = certificates.GenerateNewServiceCertificate(ctx, s, "client", certificates.CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for client: %s", err)
	}

	// Copy shared CA certificate from shared database to file on disk
	err = certificates.DumpCA(ctx, s)
	if err != nil {
		return err
	}

	// Start all the required services, and central if needed
	if enableServices.Switch {
		err = node.EnableService(ctx, s, types.SrvSwitch, nil)
		if err != nil {
			logger.Infof("Failed to enable switch")
			return err
		}
	}

	if enableServices.Central {
		err = node.EnableService(ctx, s, types.SrvCentral, nil)
		if err != nil {
			logger.Infof("Failed to enable central")
			return err
		}

		err = environment.GenerateEnvironment(ctx, s)
		if err != nil {
			return fmt.Errorf("failed to generate the daemon configuration: %w", err)
		}
	}

	if enableServices.Chassis {
		err = node.EnableService(ctx, s, types.SrvChassis, nil)
		if err != nil {
			logger.Infof("Failed to enable ovn-controller")
			return err
		}

		_, err = ovnCmd.VSCtl(
			ctx,
			s,
			"set", "open_vswitch", ".",
			fmt.Sprintf("external_ids:system-id=%s", s.Name()),
			"external_ids:ovn-encap-type=geneve",
			fmt.Sprintf("external_ids:ovn-encap-ip=%s", ovnEncapIP),
		)

		if err != nil {
			return fmt.Errorf("error configuring OVS parameters: %s", err)
		}

		err = ovnCluster.UpdateOvnControllerRemoteConfig(ctx, s)
		if err != nil {
			return err
		}
	}

	return nil
}
