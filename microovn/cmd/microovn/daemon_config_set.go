package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/spf13/cobra"
)

type cmdDaemonConfigSet struct {
	common       *CmdControl
	daemonConfig *cmdDaemonConfig
}

// Command returns definition for "microovn daemon-config set" subcommand
func (c *cmdDaemonConfigSet) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <KEY> <VALUE>",
		Short: "Set a value in the local daemon configuration",
		Args:  cobra.ExactArgs(2),
		RunE:  c.Run,
	}
	return cmd
}

// Run method is an implementation of the "microovn daemon-config set" subcommand
func (c *cmdDaemonConfigSet) Run(_ *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	if key != daemonConfigKeyFailureDomain {
		return fmt.Errorf("daemon configuration key '%s' is not settable", key)
	}

	failureDomain, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid failure-domain '%s': must be an unsigned 64-bit integer", value)
	}

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	// Restart is requested so the failure-domain is applied to dqlite
	// immediately. On a bootstrapped node this rolls the local dqlite app
	// (with a leader handoff where applicable) rather than restarting
	// microovnd.
	patch := types.DaemonConfigPatch{FailureDomain: &failureDomain}
	if err := m.UpdateDaemonConfig(context.Background(), patch, true); err != nil {
		return fmt.Errorf("failed to set daemon configuration key '%s': %w", key, err)
	}

	fmt.Printf("Successfully set daemon configuration option '%s'\n", key)
	return nil
}
