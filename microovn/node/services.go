package node

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/database"
)

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
