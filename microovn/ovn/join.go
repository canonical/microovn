package ovn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microcluster/state"
	"github.com/canonical/microovn/microovn/database"
)

// Join will join an existing OVN deployment.
func Join(s *state.State) error {
	// Make sure we don't have any other hooks firing.
	muHook.Lock()
	defer muHook.Unlock()

	// Create our storage.
	err := createPaths()
	if err != nil {
		return err
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

	// Record the new roles in the database.
	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		// Record the roles.
		_, err := database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: "switch"})
		if err != nil {
			return fmt.Errorf("Failed to record role: %w", err)
		}

		if srvCentral < 3 {
			_, err = database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: "central"})
			if err != nil {
				return fmt.Errorf("Failed to record role: %w", err)
			}
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

	// Generate the configuration.
	err = generateEnvironment(s)
	if err != nil {
		return fmt.Errorf("Failed to generate the daemon configuration: %w", err)
	}

	// Copy shared CA certificate from shared database to file on disk
	err = DumpCA(s)
	if err != nil {
		return err
	}

	// Enable OVS switch.
	err = snapStart("switch", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVS switch: %w", err)
	}

	// Enable OVN central (if needed).
	if srvCentral < 3 {
		// Generate certificate for OVN Central services
		err = GenerateNewServiceCertificate(s, "ovnnb", CertificateTypeServer)
		if err != nil {
			return fmt.Errorf("failed to generate TLS certificate for ovnnb service")
		}
		err = GenerateNewServiceCertificate(s, "ovnsb", CertificateTypeServer)
		if err != nil {
			return fmt.Errorf("failed to generate TLS certificate for ovnsb service")
		}
		err = GenerateNewServiceCertificate(s, "ovn-northd", CertificateTypeServer)
		if err != nil {
			return fmt.Errorf("failed to generate TLS certificate for ovn-northd service")
		}

		err = snapStart("central", true)
		if err != nil {
			return fmt.Errorf("Failed to start OVN central: %w", err)
		}
	}

	// Generate certificate for OVN chassis (controller)
	err = GenerateNewServiceCertificate(s, "ovn-controller", CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovn-controller service")
	}
	err = snapStart("chassis", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN chassis: %w", err)
	}

	// Enable OVN chassis.
	sbConnect, err := connectString(s, 6642)
	if err != nil {
		return fmt.Errorf("Failed to get OVN SB connect string: %w", err)
	}

	_, err = VSCtl(
		s,
		"set", "open_vswitch", ".",
		fmt.Sprintf("external_ids:system-id=%s", s.Name()),
		fmt.Sprintf("external_ids:ovn-remote=%s", sbConnect),
		"external_ids:ovn-encap-type=geneve",
		fmt.Sprintf("external_ids:ovn-encap-ip=%s", s.Address().Hostname()),
	)

	if err != nil {
		return fmt.Errorf("Error configuring OVS parameters: %s", err)
	}

	return nil
}
