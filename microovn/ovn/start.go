package ovn

import (
	"context"
	"fmt"

	"github.com/canonical/lxd/shared/logger"

	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/node"
	ovnCluster "github.com/canonical/microovn/microovn/ovn/cluster"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/environment"
	"github.com/canonical/microovn/microovn/ovn/ovsdb"
)

// Start will update the existing OVN central and OVS switch configs.
func Start(ctx context.Context, s state.State) error {
	// Skip if the database isn't ready.
	err := s.Database().IsOpen(ctx)
	if err != nil {
		logger.Warn("Skipping OVN configuration, cluster database is offline", logger.Ctx{"error": err})
		return nil
	}

	// Make sure the storage exists.
	err = environment.CreatePaths()
	if err != nil {
		return err
	}

	err = node.ActivateEnabledServices(ctx, s, true)
	if err != nil {
		return fmt.Errorf("failed to enable required services: %w", err)
	}

	// Re-generate the configuration.
	err = environment.GenerateEnvironment(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to generate the daemon configuration: %w", err)
	}

	centralActive, err := node.HasServiceActive(ctx, s, "central")
	if err != nil {
		return fmt.Errorf("failed to query local services: %w", err)
	}

	if centralActive {
		err = ovnCluster.UpdateOvnListenConfig(ctx, s)
		if err != nil {
			logger.Warnf("Failed to update OVN listening configs. There might be connectivity issues.")
		}
	}
	// Reconfigure OVS to use OVN.
	err = ovnCluster.UpdateOvnControllerRemoteConfig(ctx, s)
	if err != nil {
		return err
	}

	// If "central" services are active on this node, start two goroutines that will check if OVN database schemas
	// are up-to-date. If a schema upgrade is required, they will coordinate with other members in the cluster and
	// trigger the schema upgrade.
	//
	// Note: The upgrade functions for NB and SB databases are started in goroutines, otherwise they'd block
	// microovnd service from fully starting.
	if centralActive {
		go func() {
			err := ovsdb.UpgradeCentralDB(ctx, s, ovnCmd.OvsdbTypeSBLocal)
			if err != nil {
				logger.Errorf("Failed to perform OVN SB schema upgrade. '%s'", err)
			}
		}()

		go func() {
			err := ovsdb.UpgradeCentralDB(ctx, s, ovnCmd.OvsdbTypeNBLocal)
			if err != nil {
				logger.Errorf("Failed to perform OVN NB schema upgrade. '%s'", err)
			}
		}()
	}

	return nil
}
