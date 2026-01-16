package main

import (
	"context"

	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/spf13/cobra"
)

type cmdClusterRemove struct {
	common  *CmdControl
	cluster *cmdCluster

	flagForce bool
}

func (c *cmdClusterRemove) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <NAME>",
		Short: "Removes a server from the cluster",
		Args:  cobra.MatchAll(cobra.ExactArgs(1)),
		RunE:  c.Run,
	}

	cmd.Flags().BoolVarP(&c.flagForce, "force", "f", false, "Forcibly remove the cluster member")

	return cmd
}

func (c *cmdClusterRemove) Run(_ *cobra.Command, args []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	err = m.RemoveClusterMember(context.Background(), args[0], c.flagForce)
	if err != nil {
		return err
	}

	return nil
}
