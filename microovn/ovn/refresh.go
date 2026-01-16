package ovn

import (
	"context"
	"fmt"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v3/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/node"
	ovnCluster "github.com/canonical/microovn/microovn/ovn/cluster"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/environment"
	"github.com/canonical/microovn/microovn/snap"
)

// Refresh will update the existing OVN central and OVS switch configs.
func Refresh(ctx context.Context, s state.State) {
	err := refresh(ctx, s)
	if err != nil {
		logger.Errorf("Failed to refresh configuration: %v", err)
	}
}

func refresh(ctx context.Context, s state.State) error {
	// Make sure we don't have any other hooks firing.
	muHook.Lock()
	defer muHook.Unlock()

	// Create our storage.
	err := environment.CreatePaths()
	if err != nil {
		return err
	}

	// Query existing local services.
	hasCentral, err := node.HasServiceActive(ctx, s, types.SrvCentral)
	if err != nil {
		return err
	}

	hasSwitch, err := node.HasServiceActive(ctx, s, types.SrvSwitch)
	if err != nil {
		return err
	}

	hasChassis, err := node.HasServiceActive(ctx, s, types.SrvChassis)
	if err != nil {
		return err
	}

	// Generate the configuration.
	err = environment.GenerateEnvironment(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to generate the daemon configuration: %w", err)
	}

	// Restart OVN Northd service to account for NB/SB cluster changes.
	if hasCentral {
		err = snap.Restart(ctx, "ovn-northd")
		if err != nil {
			return fmt.Errorf("failed to restart OVN northd: %w", err)
		}
	}

	// Enable OVN chassis.
	if hasSwitch {
		err = ovnCluster.UpdateOvnControllerRemoteConfig(ctx, s)
		if err != nil {
			return err
		}
	}

	// In the event when we are re-bootstrapping central cluster, we need to
	// clear the previous cluster's state from the controller. This is a less
	// invasive alternative to controller restart.
	if hasChassis {
		_, err = ovnCmd.AppCtl(
			ctx,
			s,
			"ovn-controller",
			"sb-cluster-state-reset",
		)
		if err != nil {
			return fmt.Errorf("failed to reset OVN chassis cluster state")
		}
	}

	return nil
}
