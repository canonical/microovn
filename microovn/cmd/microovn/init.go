package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/canonical/lxd/lxd/util"
	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/microcluster/microcluster"
	"github.com/spf13/cobra"
)

type cmdInit struct {
	common *CmdControl

	flagBootstrap bool
	flagToken     string
}

func (c *cmdInit) Command() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive configuration of MicroOVN",
		RunE:  c.Run,
	}

	return cmd
}

func (c *cmdInit) Run(cmd *cobra.Command, args []string) error {
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

		if wantsBootstrap {
			mode = "bootstrap"

			// Offer overriding the name.
			hostName, err = c.common.asker.AskString(fmt.Sprintf("Please choose a name for this system [default=%s]: ", hostName), hostName, nil)
			if err != nil {
				return err
			}

			// Bootstrap the cluster.
			err = m.NewCluster(context.Background(), hostName, address, nil)
			if err != nil {
				return err
			}
		} else {
			mode = "join"

			token, err := c.common.asker.AskString("Please enter your join token: ", "", nil)
			if err != nil {
				return err
			}

			err = m.JoinCluster(context.Background(), hostName, address, token, nil)
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
				tokenName, err := c.common.asker.AskString("What's the name of the new MicroOVN server? (empty to exit): ", "", func(input string) error { return nil })
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
