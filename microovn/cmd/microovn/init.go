package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/canonical/lxd/lxd/util"
	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/lxd/shared/validate"
	"github.com/canonical/microcluster/v2/microcluster"
	microovnAPI "github.com/canonical/microovn/microovn/api"
	"github.com/spf13/cobra"
)

type cmdInit struct {
	common *CmdControl
}

func (c *cmdInit) Command() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive configuration of MicroOVN",
		RunE:  c.Run,
	}

	return cmd
}

func (c *cmdInit) wantsCustomEncapsulationIP() (string, string, error) {
	wantsCustomEncapsulationIP, err := c.common.asker.AskBool("Would you like to define a custom encapsulation IP address for this member? (yes/no) [default=no]: ", "no")
	if err != nil {
		return "", "", err
	}

	if wantsCustomEncapsulationIP {
		encapIP, err := c.common.asker.AskString("Please enter the custom encapsulation IP address for this member: ", "", validate.Required(validate.IsNetworkAddress))
		if err != nil {
			return "", "", err
		}

		return "ovn-encap-ip", encapIP, nil
	}

	return "", "", nil
}

func (c *cmdInit) Run(_ *cobra.Command, _ []string) error {
	// Connect to the daemon.
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir, Verbose: c.common.FlagLogVerbose, Debug: c.common.FlagLogDebug})
	if err != nil {
		return err
	}

	lc, err := m.LocalClient()
	if err != nil {
		return err
	}

	// Check if already initialized.
	_, err = lc.GetClusterMembers(context.Background())
	isUninitialized := err != nil && api.StatusErrorCheck(err, http.StatusServiceUnavailable)
	if err != nil && !isUninitialized {
		return err
	}

	// User interaction.
	mode := "existing"
	customEncapsulationIPSupported := shared.ValueInSlice("custom_encapsulation_ip", microovnAPI.Extensions())

	if isUninitialized {
		// Get system name.
		hostName, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("Failed to retrieve system hostname: %w", err)
		}

		// Get system address.
		address := util.NetworkInterfaceAddress()
		address, err = c.common.asker.AskString(fmt.Sprintf("Please choose the address MicroOVN will be listening on [default=%s]: ", address), address, nil)
		if err != nil {
			return err
		}
		address = util.CanonicalNetworkAddress(address, 6443)

		wantsBootstrap, err := c.common.asker.AskBool("Would you like to create a new MicroOVN cluster? (yes/no) [default=no]: ", "no")
		if err != nil {
			return err
		}

		var optionalConfig map[string]string
		if wantsBootstrap {
			mode = "bootstrap"

			// Offer overriding the name.
			hostName, err = c.common.asker.AskString(fmt.Sprintf("Please choose a name for this system [default=%s]: ", hostName), hostName, nil)
			if err != nil {
				return err
			}

			if customEncapsulationIPSupported {
				key, encapIP, err := c.wantsCustomEncapsulationIP()
				if err != nil {
					return err
				}

				if key != "" && encapIP != "" {
					optionalConfig = map[string]string{key: encapIP}
				}
			}

			// Bootstrap the cluster.
			err = m.NewCluster(context.Background(), hostName, address, optionalConfig)
			if err != nil {
				return err
			}
		} else {
			mode = "join"

			token, err := c.common.asker.AskString("Please enter your join token: ", "", nil)
			if err != nil {
				return err
			}

			if customEncapsulationIPSupported {
				// Register a potential custom encapsulation IP for other systems,
				// so that when they will join the cluster, their encapsulation IP
				// for the Geneve tunnel will be automatically configured.
				key, encapIP, err := c.wantsCustomEncapsulationIP()
				if err != nil {
					return err
				}

				if key != "" && encapIP != "" {
					optionalConfig = map[string]string{key: encapIP}
				}
			}

			err = m.JoinCluster(context.Background(), hostName, address, token, optionalConfig)
			if err != nil {
				return err
			}
		}
	} else {
		fmt.Printf("MicroOVN has already been initialized.\n\n")
	}

	// Add additional servers.
	if mode != "join" {
		wantsMachines, err := c.common.asker.AskBool("Would you like to add additional servers to the cluster? (yes/no) [default=no]: ", "no")
		if err != nil {
			return err
		}

		if wantsMachines {
			for {
				tokenName, err := c.common.asker.AskString("What's the name of the new MicroOVN server? (empty to exit): ", "", func(_ string) error { return nil })
				if err != nil {
					return err
				}

				if tokenName == "" {
					break
				}

				// Issue the token.
				token, err := m.NewJoinToken(context.Background(), tokenName)
				if err != nil {
					return err
				}

				fmt.Println(token)
			}
		}
	}

	return nil
}
