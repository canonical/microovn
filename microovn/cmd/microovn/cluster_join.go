package main

import (
	"context"
	"fmt"
	"os"

	"github.com/canonical/lxd/lxd/util"
	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/spf13/cobra"
)

type cmdClusterJoin struct {
	common  *CmdControl
	cluster *cmdCluster
}

func (c *cmdClusterJoin) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join <TOKEN>",
		Short: "Joins an existing cluster",
		Args:  cobra.MatchAll(cobra.ExactArgs(1)),
		RunE:  c.Run,
	}

	return cmd
}

func (c *cmdClusterJoin) Run(_ *cobra.Command, args []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return fmt.Errorf("unable to configure microcluster: %w", err)
	}

	// Get system hostname.
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to retrieve system hostname: %w", err)
	}

	// Get system address.
	address := util.NetworkInterfaceAddress()
	address = util.CanonicalNetworkAddress(address, DefaultMicroClusterPort)

	return m.JoinCluster(context.Background(), hostname, address, args[0], nil)
}
