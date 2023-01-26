package ovn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microovn/microovn/database"

	"github.com/canonical/microcluster/state"
)

// Join will join an existing OVN deployment.
func Join(s *state.State) error {
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

	// Query existing core services.
	srvCentral := 0

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
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

	// Add additional services as required.
	services := []string{"chassis"}

	if srvCentral < 3 {
		services = append(services, "central")
	}

	// Update the database.
	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		// Record the roles.
		for _, service := range services {
			_, err := database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: service})
			if err != nil {
				return fmt.Errorf("Failed to record role: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
