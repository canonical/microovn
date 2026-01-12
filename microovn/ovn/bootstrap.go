package ovn

import (
	"context"
	"fmt"
	"os"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/node"
	"github.com/canonical/microovn/microovn/ovn/certificates"
	ovnCluster "github.com/canonical/microovn/microovn/ovn/cluster"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/dpu"
	"github.com/canonical/microovn/microovn/ovn/environment"
)

// Bootstrap will initialize a new OVN deployment.
func Bootstrap(ctx context.Context, s state.State, initConfig map[string]string) error {
	// Make sure we don't have any other hooks firing.
	muHook.Lock()
	defer muHook.Unlock()

	// Create our storage.
	err := environment.CreatePaths()
	if err != nil {
		return err
	}

	// The default behavior on bootstrap is to enable all services
	enableServices := requestedServices{
		Central: true,
		Chassis: true,
		Switch:  true,
	}

	// Parse custom bootstrap options from initConfig
	ovnEncapIP := s.Address().Hostname()
	var certPem []byte
	var keyPem []byte
	for k, v := range initConfig {
		// Configure OVS to either use a custom encapsulation IP for the geneve tunel
		// or the hostname of the node.
		if k == "ovn-encap-ip" {
			ovnEncapIP = v
			continue
		}

		// Retrieve the CA certificate and private key path if the user supplied them during init
		if k == "ovn-ca-cert" {
			certPem, err = os.ReadFile(v)
			if err != nil {
				return fmt.Errorf("failed to read CA certificate: %w", err)
			}
			continue
		}
		if k == "ovn-ca-key" {
			keyPem, err = os.ReadFile(v)
			if err != nil {
				return fmt.Errorf("failed to read CA private key: %w", err)
			}
			continue
		}

		// Get requested services
		if k == "ovn-services" {
			if v != "auto" {
				enableServices, err = newRequestedServices(v)
				if err != nil {
					return fmt.Errorf("failed to parse requested services: %w", err)
				}
			}
		}
	}

	// Generate CA certificate and key
	if len(certPem) != 0 && len(keyPem) != 0 {
		_, err = certificates.SetNewCACertificate(ctx, s, string(certPem), string(keyPem))
		if err != nil {
			return err
		}
	} else {
		_, err = certificates.GenerateNewCACertificate(ctx, s)
		if err != nil {
			return err
		}
	}

	err = certificates.DumpCA(ctx, s)
	if err != nil {
		return err
	}

	// Generate the configuration.
	err = environment.GenerateEnvironment(ctx, s)
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
	if enableServices.Switch {
		err = node.EnableService(ctx, s, types.SrvSwitch, nil)
		if err != nil {
			logger.Infof("Failed to enable switch")
			return err
		}
	}

	if enableServices.Central {
		err = node.EnableService(ctx, s, types.SrvCentral, nil)
		if err != nil {
			logger.Infof("Failed to enable central")
			return err
		}

		err = environment.GenerateEnvironment(ctx, s)
		if err != nil {
			return fmt.Errorf("failed to generate the daemon configuration: %w", err)
		}

		err = ovnCluster.UpdateOvnListenConfig(ctx, s)
		if err != nil {
			return err
		}
	}

	if enableServices.Chassis {
		err = node.EnableService(ctx, s, types.SrvChassis, nil)
		if err != nil {
			logger.Infof("Failed to enable ovn-controller")
			return err
		}

		_, err = ovnCmd.VSCtl(
			ctx,
			s,
			"set", "open_vswitch", ".",
			fmt.Sprintf("external_ids:system-id=%s", s.Name()),
			"external_ids:ovn-encap-type=geneve",
			fmt.Sprintf("external_ids:ovn-encap-ip=%s", ovnEncapIP),
		)

		if err != nil {
			return fmt.Errorf("error configuring OVS parameters: %s", err)
		}

		err = ovnCluster.UpdateOvnControllerRemoteConfig(ctx, s)
		if err != nil {
			return err
		}
	}

	err = dpu.DPUSetup(ctx, s)
	if err != nil {
		return err
	}

	return nil
}
