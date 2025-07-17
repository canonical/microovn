package main

import (
	"context"
	"fmt"

	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/canonical/microovn/microovn/client"
	"github.com/spf13/cobra"
)

type cmdConfigSet struct {
	common *CmdControl
	config *cmdConfig
}

// Command returns definition for "microovn config set" subcommand
func (c *cmdConfigSet) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <KEY> <VALUE>",
		Short: "Set or update configuration value",
		Args:  cobra.ExactArgs(2),
		RunE:  c.Run,
	}
	return cmd
}

// Run method is an implementation of the "microovn config set" subcommand
func (c *cmdConfigSet) Run(_ *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	response, err := client.SetConfig(context.Background(), cli, key, value)

	if err != nil {
		return fmt.Errorf("failed to set config option '%s': %s", key, err)
	}

	if response.Error != "" {
		return fmt.Errorf("failed to set config option '%s': %s", key, response.Error)
	}

	fmt.Printf("Successfully set config option '%s'\n", key)
	return nil
}
