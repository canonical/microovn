package main

import (
	"context"
	"time"

	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/spf13/cobra"
)

type cmdWaitReady struct {
	common *CmdControl

	flagTimeout int
}

func (c *cmdWaitReady) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "waitready",
		Short: "Wait for the daemon to be ready to process requests",
		RunE:  c.Run,
		Args:  cobra.MatchAll(cobra.ExactArgs(0)),
	}

	cmd.Flags().IntVarP(&c.flagTimeout, "timeout", "t", 0,
		"Number of seconds to wait before giving up")

	return cmd
}

func (c *cmdWaitReady) Run(cmd *cobra.Command, _ []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	ctx, cancel := cmd.Context(), func() {}
	if c.flagTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(c.flagTimeout)*time.Second)
	}

	defer cancel()

	return m.Ready(ctx)
}
