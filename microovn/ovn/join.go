package ovn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/node"
	"github.com/canonical/microovn/microovn/ovn/certificates"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/snap"
)

// Join will join an existing OVN deployment.
func Join(ctx context.Context, s state.State, initConfig map[string]string) error {
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
	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
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
	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
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
	err = generateEnvironment(ctx, s)
	if err != nil {
		return fmt.Errorf("Failed to generate the daemon configuration: %w", err)
	}

	// Generate client certificate for managing OVN Central services
	// Note that we intentially use a sever type certificate here due to
	// all OVS-based programs ability to specify active or passive (listen)
	// connection types.
	err = certificates.GenerateNewServiceCertificate(ctx, s, "client", certificates.CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for client: %s", err)
	}

	// Copy shared CA certificate from shared database to file on disk
	err = certificates.DumpCA(ctx, s)
	if err != nil {
		return err
	}

	// Enable OVS switch.
	err = snap.Start("switch", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVS switch: %w", err)
	}

	// Enable OVN central (if needed).
	if srvCentral < 3 {
		err = node.JoinCentral(ctx, s)
		if err != nil {
			return fmt.Errorf("Failed to start OVN central: %w", err)
		}
	}

	// Generate certificate for OVN chassis (controller)
	err = certificates.GenerateNewServiceCertificate(ctx, s, "ovn-controller", certificates.CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovn-controller service")
	}
	err = snap.Start("chassis", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN chassis: %w", err)
	}

	// Enable OVN chassis.
	sbConnect, _, err := environmentString(ctx, s, 6642)
	if err != nil {
		return fmt.Errorf("Failed to get OVN SB connect string: %w", err)
	}

	// A custom encapsulation IP address can also be directly passed as an initConfig parameter.
	// This block is typically executed by a `microovn cluster init` or by an external project
	// triggering this join hook.
	var ovnEncapIP string
	for k, v := range initConfig {
		if k == "ovn-encap-ip" {
			ovnEncapIP = v
			break
		}
	}

	if ovnEncapIP == "" {
		ovnEncapIP = s.Address().Hostname()
	}

	_, err = ovnCmd.VSCtl(
		ctx,
		s,
		"set", "open_vswitch", ".",
		fmt.Sprintf("external_ids:system-id=%s", s.Name()),
		fmt.Sprintf("external_ids:ovn-remote=%s", sbConnect),
		"external_ids:ovn-encap-type=geneve",
		fmt.Sprintf("external_ids:ovn-encap-ip=%s", ovnEncapIP),
	)

	if err != nil {
		return fmt.Errorf("Error configuring OVS parameters: %s", err)
	}

	return nil
}
