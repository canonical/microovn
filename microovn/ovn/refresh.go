package ovn

import (
	"fmt"

	"github.com/canonical/lxd/shared/logger"
	"github.com/pkg/errors"

	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/node"
	"github.com/canonical/microovn/microovn/snap"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
)

// Refresh will update the existing OVN central and OVS switch configs.
func Refresh(s *state.State) error {
	// Don't block the caller on a refresh as we may build a backlog.
	go func(s *state.State) {
		err := refresh(s)
		if err != nil {
			logger.Errorf("Failed to refresh configuration: %v", err)
		}
	}(s)

	return nil
}

func refresh(s *state.State) error {
	// Make sure we don't have any other hooks firing.
	muHook.Lock()
	defer muHook.Unlock()

	// Create our storage.
	err := createPaths()
	if err != nil {
		return err
	}

	// Query existing local services.
	hasCentral, err := node.HasServiceActive(s, "central")
	if err != nil {
		return err
	}

	hasSwitch, err := node.HasServiceActive(s, "switch")
	if err != nil {
		return err
	}

	// Generate the configuration.
	err = generateEnvironment(s)
	if err != nil {
		return fmt.Errorf("Failed to generate the daemon configuration: %w", err)
	}

	// Restart OVN Northd service to account for NB/SB cluster changes.
	if hasCentral {
		err = snap.Restart("ovn-northd")
		if err != nil {
			return fmt.Errorf("Failed to restart OVN northd: %w", err)
		}
	}

	// Enable OVN chassis.
	if hasSwitch {
		// Reconfigure OVS to use OVN.
		sbConnect, _, err := environmentString(s, 6642)
		if err != nil {
			return fmt.Errorf("Failed to get OVN SB connect string: %w", err)
		}

		_, err = ovnCmd.VSCtl(
			s,
			"set", "open_vswitch", ".",
			fmt.Sprintf("external_ids:ovn-remote=%s", sbConnect),
		)

		if err != nil {
			return fmt.Errorf("Failed to update OVS's 'ovn-remote' configuration")
		}
	}

	return nil
}

func updateOvnListenConfig(s *state.State) error {
	nbDB, err := ovnCmd.NewOvsdbSpec(ovnCmd.OvsdbTypeNBLocal)
	if err != nil {
		return fmt.Errorf("Failed to get path to OVN NB database socket: %w", err)
	}
	sbDB, err := ovnCmd.NewOvsdbSpec(ovnCmd.OvsdbTypeSBLocal)
	if err != nil {
		return fmt.Errorf("Failed to get path to OVN SB database socket: %w", err)
	}

	protocol := networkProtocol(s)
	_, err = ovnCmd.NBCtl(
		s,
		"--no-leader-only",
		fmt.Sprintf("--db=%s", nbDB.SocketURL),
		"set-connection",
		fmt.Sprintf("p%s:6641:[::]", protocol),
	)
	if err != nil {
		return errors.Errorf("Error setting ovn NB connection string: %s", err)
	}

	_, err = ovnCmd.SBCtl(
		s,
		"--no-leader-only",
		fmt.Sprintf("--db=%s", sbDB.SocketURL),
		"set-connection",
		fmt.Sprintf("p%s:6642:[::]", protocol),
	)
	if err != nil {
		return errors.Errorf("Error setting ovn SB connection string: %s", err)
	}

	return nil
}
