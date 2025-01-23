package main

import (
	"context"
	"fmt"
	"os"

	"github.com/canonical/lxd/lxd/util"
	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"
)

type cmdClusterBootstrap struct {
	common  *CmdControl
	cluster *cmdCluster
}

func (c *cmdClusterBootstrap) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Sets up a new cluster",
		RunE:  c.Run,
	}

	return cmd
}

func (c *cmdClusterBootstrap) Run(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		fmt.Println("Incorrect number of arguments.")
		_ = cmd.Help()
		return fmt.Errorf("invalid arguments")
	}

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return fmt.Errorf("Unable to configure MicroOVN: %w", err)
	}

	// Get system hostname.
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("Failed to retrieve system hostname: %w", err)
	}

	// Get system address.
	address := util.NetworkInterfaceAddress()
	address = util.CanonicalNetworkAddress(address, 6443)

	return m.NewCluster(context.Background(), hostname, address, nil)
}
