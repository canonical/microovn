package main

import (
	"context"
	"fmt"
	"os"

	"github.com/canonical/lxd/lxd/util"
	"github.com/canonical/microcluster/v3/microcluster"
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
		Args:  cobra.MatchAll(cobra.ExactArgs(0)),
		RunE:  c.Run,
	}

	return cmd
}

func (c *cmdClusterBootstrap) Run(_ *cobra.Command, _ []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return fmt.Errorf("unable to configure microovn: %w", err)
	}

	// Get system hostname.
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to retrieve system hostname: %w", err)
	}

	// Get system address.
	address := util.NetworkInterfaceAddress()
	address = util.CanonicalNetworkAddress(address, DefaultMicroClusterPort)

	return m.NewCluster(context.Background(), hostname, address, nil)
}
