package ovn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/database"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/snap"
)

// Join will join an existing OVN deployment.
func Join(s *state.State, initConfig map[string]string) error {
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

	// Generate client certificate for managing OVN Central services
	// Note that we intentially use a sever type certificate here due to
	// all OVS-based programs ability to specify active or passive (listen)
	// connection types.
	err = GenerateNewServiceCertificate(s, "client", CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for client: %s", err)
	}

	// Copy shared CA certificate from shared database to file on disk
	err = DumpCA(s)
	if err != nil {
		return err
	}

	// Enable OVS switch.
	err = snap.SnapStart("switch", true)
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

		err = snap.SnapStart("ovn-ovsdb-server-nb", true)
		if err != nil {
			return fmt.Errorf("Failed to start OVN NB: %w", err)
		}

		err = snap.SnapStart("ovn-ovsdb-server-sb", true)
		if err != nil {
			return fmt.Errorf("Failed to start OVN SB: %w", err)
		}

		err = snap.SnapStart("ovn-northd", true)
		if err != nil {
			return fmt.Errorf("Failed to start OVN northd: %w", err)
		}
	}

	// Generate certificate for OVN chassis (controller)
	err = GenerateNewServiceCertificate(s, "ovn-controller", CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovn-controller service")
	}
	err = snap.SnapStart("chassis", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN chassis: %w", err)
	}

	// Enable OVN chassis.
	sbConnect, _, err := environmentString(s, 6642)
	if err != nil {
		return fmt.Errorf("Failed to get OVN SB connect string: %w", err)
	}

	// A custom encapsulation IP address can also be directly passed as an initConfig parameter.
	// This block is typically executed by a `microovn cluster init` or by an external project
	// triggering this join hook.
	var ovnEncapIp string
	for k, v := range initConfig {
		if k == "ovn-encap-ip" {
			ovnEncapIp = v
			break
		}
	}

	if ovnEncapIp == "" {
		ovnEncapIp = s.Address().Hostname()
	}

	_, err = ovnCmd.VSCtl(
		s,
		"set", "open_vswitch", ".",
		fmt.Sprintf("external_ids:system-id=%s", s.Name()),
		fmt.Sprintf("external_ids:ovn-remote=%s", sbConnect),
		"external_ids:ovn-encap-type=geneve",
		fmt.Sprintf("external_ids:ovn-encap-ip=%s", ovnEncapIp),
	)

	if err != nil {
		return fmt.Errorf("Error configuring OVS parameters: %s", err)
	}

	return nil
}
