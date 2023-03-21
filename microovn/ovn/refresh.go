package ovn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microovn/microovn/database"
	"github.com/lxc/lxd/shared"

	"github.com/canonical/microcluster/state"
)

// Refresh will update the existing OVN central and OVS switch configs.
func Refresh(s *state.State) error {
	// Create our storage.
	err := createPaths()
	if err != nil {
		return err
	}

	// Query existing local services.
	hasCentral := false
	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		// Check if we have the central service.
		name := s.Name()
		services, err := database.GetServices(ctx, tx, database.ServiceFilter{Member: &name})
		if err != nil {
			return err
		}

		for _, srv := range services {
			if srv.Service == "central" {
				hasCentral = true
				break
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Generate the configuration.
	err = generateEnvironment(s)
	if err != nil {
		return fmt.Errorf("Failed to generate the daemon configuration: %w", err)
	}

	// Enable OVN central (if needed).
	if hasCentral {
		err = snapRestart("central")
		if err != nil {
			return fmt.Errorf("Failed to start OVN central: %w", err)
		}
	}

	// Enable OVN chassis.
	err = snapRestart("chassis")
	if err != nil {
		return fmt.Errorf("Failed to restart OVN chassis: %w", err)
	}

	// Reconfigure OVS to use OVN.
	sbConnect, err := connectString(s, 6642)
	if err != nil {
		return fmt.Errorf("Failed to get OVN SB connect string: %w", err)
	}

	_, err = shared.RunCommand(
		"ovs-vsctl",
		"set", "open_vswitch", ".",
		fmt.Sprintf("external_ids:ovn-remote=%s", sbConnect),
	)

	return nil
}
