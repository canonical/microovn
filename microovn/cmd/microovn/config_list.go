package main

import (
	"fmt"

	"github.com/canonical/microovn/microovn/api/config"
	"github.com/spf13/cobra"
)

type cmdConfigList struct {
	common *CmdControl
	config *cmdConfig
}

// Command returns definition for "microovn config list" subcommand
func (c *cmdConfigList) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List allowed configuration keys",
		RunE:  c.Run,
	}
	return cmd
}

// Run method is an implementation of the "microovn config list" subcommand
func (c *cmdConfigList) Run(_ *cobra.Command, args []string) error {
	fmt.Println("Available options:")
	for _, keySpec := range config.AllowedConfigKeys {
		fmt.Println(keySpec.Key)
	}
	return nil
}
