package ovn

import (
	"context"
	"fmt"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/node"
	"github.com/canonical/microovn/microovn/ovn/certificates"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
)

// Bootstrap will initialize a new OVN deployment.
func Bootstrap(ctx context.Context, s state.State, initConfig map[string]string) error {
	// Make sure we don't have any other hooks firing.
	muHook.Lock()
	defer muHook.Unlock()

	// Create our storage.
	err := createPaths()
	if err != nil {
		return err
	}

	// Generate CA certificate and key
	err = certificates.GenerateNewCACertificate(ctx, s)
	if err != nil {
		return err
	}

	err = certificates.DumpCA(ctx, s)
	if err != nil {
		return err
	}

	// Generate the configuration.
	err = generateEnvironment(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to generate the daemon configuration: %w", err)
	}

	// Generate client certificate for managing OVN Central services
	// Note that we intentially use a sever type certificate here due to
	// all OVS-based programs ability to specify active or passive (listen)
	// connection types.
	err = certificates.GenerateNewServiceCertificate(ctx, s, "client", certificates.CertificateTypeServer)
	if err != nil {
		return fmt.Errorf("failed to generate TLS certificate for client: %s", err)
	}

	// Start all the required services

	err = node.EnableService(ctx, s, types.SrvSwitch)
	if err != nil {
		logger.Infof("Failed to enable switch")
		return err
	}

	err = node.EnableService(ctx, s, types.SrvCentral)
	if err != nil {
		logger.Infof("Failed to enable central")
		return err
	}

	err = generateEnvironment(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to generate the daemon configuration: %w", err)
	}

	err = node.EnableService(ctx, s, types.SrvChassis)
	if err != nil {
		logger.Infof("Failed to enable switch")
		return err
	}

	// Configure OVS to use OVN.
	sbConnect, _, err := environmentString(ctx, s, 6642)
	if err != nil {
		return fmt.Errorf("failed to get OVN SB connect string: %w", err)
	}

	err = updateOvnListenConfig(ctx, s)
	if err != nil {
		return err
	}

	// Configure OVS to either use a custom encapsulation IP for the geneve tunel
	// or the hostname of the node.
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
		return fmt.Errorf("error configuring OVS parameters: %s", err)
	}

	return nil
}
