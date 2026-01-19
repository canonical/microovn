package main

import (
	"context"
	"fmt"
	
  "github.com/canonical/microcluster/v3/microcluster"
  "github.com/canonical/microovn/microovn/api/config"
	"github.com/canonical/microovn/microovn/client"
	"github.com/spf13/cobra"
)

type cmdConfigGet struct {
	common *CmdControl
	config *cmdConfig
}

// Command returns definition for "microovn config get" subcommand
func (c *cmdConfigGet) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <KEY>",
		Short: "Get value of the configuration option, use list to see available options",
		Args:  cobra.ExactArgs(1),
		RunE:  c.Run,
	}
	return cmd
}

// Run method is an implementation of the "microovn config get" subcommand
func (c *cmdConfigGet) Run(_ *cobra.Command, args []string) error {
	key := args[0]

	if key == "list" {
		fmt.Println("Available options")
		for _, keySpec := range config.AllowedConfigKeys {
			fmt.Println(keySpec.Key)
		}
		return nil
	}

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	response, err := client.GetConfig(context.Background(), cli, key)

	if err != nil {
		return fmt.Errorf("failed to get config option '%s': %s", key, err)
	}

	if response.Error != "" {
		return fmt.Errorf("failed to get config option '%s': %s", key, response.Error)
	}

	if response.IsSet {
		fmt.Println(response.Value)
	}
	return nil
}
