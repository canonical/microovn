package ovn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microovn/microovn/database"

	"github.com/canonical/microcluster/state"
)

// Bootstrap will initialize a new OVN deployment.
func Bootstrap(s *state.State) error {
	// Create our storage.
	err := createPaths()
	if err != nil {
		return err
	}

	// Enable OVS switch.
	err = snapStart("switch", true)
	if err != nil {
		return fmt.Errorf("Failed to start monitor: %w", err)
	}

	// Enable OVN chassis.
	err = snapStart("chassis", true)
	if err != nil {
		return fmt.Errorf("Failed to start monitor: %w", err)
	}

	// Update the database.
	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		// Record the roles.
		_, err := database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: "central"})
		if err != nil {
			return fmt.Errorf("Failed to record role: %w", err)
		}

		_, err = database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: "chassis"})
		if err != nil {
			return fmt.Errorf("Failed to record role: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
