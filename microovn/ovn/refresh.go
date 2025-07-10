package ovn

import (
	"context"
	"fmt"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/node"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/environment"
	"github.com/canonical/microovn/microovn/snap"
)

// Refresh will update the existing OVN central and OVS switch configs.
func Refresh(shutdownCtx context.Context, _ context.Context, s state.State) error {
	// Don't block the caller on a refresh as we may build a backlog.
	go func(ctx context.Context, s state.State) {
		err := refresh(ctx, s)
		if err != nil {
			logger.Errorf("Failed to refresh configuration: %v", err)
		}
	}(shutdownCtx, s)

	return nil
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

	// Generate the configuration.
	err = environment.GenerateEnvironment(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to generate the daemon configuration: %w", err)
	}

	// Restart OVN Northd service to account for NB/SB cluster changes.
	if hasCentral {
		err = snap.Restart("ovn-northd")
		if err != nil {
			return fmt.Errorf("failed to restart OVN northd: %w", err)
		}
	}

	// Enable OVN chassis.
	if hasSwitch {
		// Reconfigure OVS to use OVN.
		sbConnect, _, err := environment.ConnectionString(ctx, s, 6642)
		if err != nil {
			return fmt.Errorf("failed to get OVN SB connect string: %w", err)
		}

		_, err = ovnCmd.VSCtl(
			ctx,
			s,
			"set", "open_vswitch", ".",
			fmt.Sprintf("external_ids:ovn-remote=%s", sbConnect),
		)

		if err != nil {
			return fmt.Errorf("failed to update OVS's 'ovn-remote' configuration")
		}
	}

	return nil
}
