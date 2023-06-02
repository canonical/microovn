package ovn

import (
	"fmt"

	"github.com/canonical/microcluster/state"
)

// Start will update the existing OVN central and OVS switch configs.
func Start(s *state.State) error {
	// Skip if the database isn't ready.
	if !s.Database.IsOpen() {
		return nil
	}

	// Make sure the storage exists.
	err := createPaths()
	if err != nil {
		return err
	}

	// Re-generate the configuration.
	err = generateEnvironment(s)
	if err != nil {
		return fmt.Errorf("Failed to generate the daemon configuration: %w", err)
	}

	// Reconfigure OVS to use OVN.
	sbConnect, err := connectString(s, 6642)
	if err != nil {
		return fmt.Errorf("Failed to get OVN SB connect string: %w", err)
	}

	_, err = VSCtl(
		s,
		"set", "open_vswitch", ".",
		fmt.Sprintf("external_ids:ovn-remote=%s", sbConnect),
	)

	if err != nil {
		return fmt.Errorf("Failed to update OVS's 'ovn-remote' configuration")
	}

	return nil
}
