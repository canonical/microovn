package main

import (
	"context"
	"fmt"

	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/canonical/microovn/microovn/client"
	"github.com/spf13/cobra"
)

type cmdConfigDelete struct {
	common *CmdControl
	config *cmdConfig
}

// Command returns definition for "microovn config delete" subcommand
func (c *cmdConfigDelete) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <KEY>",
		Short: "Remove configuration value",
		Args:  cobra.ExactArgs(1),
		RunE:  c.Run,
	}
	return cmd
}

// Run method is an implementation of the "microovn config delete" subcommand
func (c *cmdConfigDelete) Run(_ *cobra.Command, args []string) error {
	key := args[0]

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	response, err := client.DeleteConfig(context.Background(), cli, key)

	if err != nil {
		return fmt.Errorf("failed to delete config option '%s': %s", key, err)
	}

	if response.Error != "" {
		return fmt.Errorf("failed to delete config option '%s': %s", key, response.Error)
	}

	fmt.Printf("Successfully deleted config option '%s'\n", key)
	return nil
}
