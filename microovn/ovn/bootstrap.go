package ovn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/ovn/paths"
)

// Bootstrap will initialize a new OVN deployment.
func Bootstrap(s *state.State) error {
	// Make sure we don't have any other hooks firing.
	muHook.Lock()
	defer muHook.Unlock()

	// Create our storage.
	err := createPaths()
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

		_, err = database.CreateService(ctx, tx, database.Service{Member: s.Name(), Service: "central"})
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

	// Generate CA certificate and key
	err = GenerateNewCACertificate(s)
	if err != nil {
		return err
	}

	err = dumpCA(s)
	if err != nil {
		return err
	}

	// Generate the configuration.
	err = generateEnvironment(s)
	if err != nil {
		return fmt.Errorf("Failed to generate the daemon configuration: %w", err)
	}

	// Enable OVS switch.
	err = snapStart("switch", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVS switch: %w", err)
	}

	// Generate certificate for OVN Central services
	nbCertPath, nbKeyPath := paths.PkiOvnNbCertFiles()
	sbCertPath, sbKeyPath := paths.PkiOvnSbCertFiles()
	northdCertPath, northdKeyPath := paths.PkiOvnNorthdCertFiles()
	err = GenerateNewServiceCertificate(s, "ovnnb", CertificateTypeServer, nbCertPath, nbKeyPath)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovnnb service: %s", err)
	}
	err = GenerateNewServiceCertificate(s, "ovnsb", CertificateTypeServer, sbCertPath, sbKeyPath)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovnsb service: %s", err)
	}
	err = GenerateNewServiceCertificate(s, "ovn-northd", CertificateTypeServer, northdCertPath, northdKeyPath)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovn-northd service: %s", err)
	}

	// Enable OVN central.
	err = snapStart("central", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN central: %w", err)
	}

	// Generate certificate for OVN chassis (controller)
	ctlCertPath, ctlKeyPath := paths.PkiOvnControllerCertFiles()
	err = GenerateNewServiceCertificate(s, "ovn-controller", CertificateTypeServer, ctlCertPath, ctlKeyPath)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for ovn-controller service: %s", err)
	}

	// Enable OVN chassis.
	err = snapStart("chassis", true)
	if err != nil {
		return fmt.Errorf("Failed to start OVN chassis: %w", err)
	}

	// Configure OVS to use OVN.
	sbConnect, err := connectString(s, 6642)
	if err != nil {
		return fmt.Errorf("Failed to get OVN SB connect string: %w", err)
	}

	nbDB, err := GetOvsdbLocalPath(OvsdbTypeNBLocal)
	if err != nil {
		return fmt.Errorf("Failed to get path to OVN NB database socket: %w", err)
	}
	sbDB, err := GetOvsdbLocalPath(OvsdbTypeSBLocal)
	if err != nil {
		return fmt.Errorf("Failed to get path to OVN SB database socket: %w", err)
	}

	_, err = NBCtl(
		s,
		fmt.Sprintf("--db=unix:%s", nbDB),
		"set-connection",
		"pssl:6641:[::]",
	)
	if err != nil {
		return fmt.Errorf("Error setting ovn NB connection string: %s", err)
	}

	_, err = SBCtl(
		s,
		fmt.Sprintf("--db=unix:%s", sbDB),
		"set-connection",
		"pssl:6642:[::]",
	)
	if err != nil {
		return fmt.Errorf("Error setting ovn SB connection string: %s", err)
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
