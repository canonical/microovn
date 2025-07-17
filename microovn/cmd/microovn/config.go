package main

import (
	"github.com/spf13/cobra"
)

type cmdConfig struct {
	common *CmdControl
}

// Command returns definition for "microovn config" subcommand
func (c *cmdConfig) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage MicroOVN configuration",
	}

	configSetCmd := &cmdConfigSet{common: c.common, config: c}
	cmd.AddCommand(configSetCmd.Command())

	configGetCmd := &cmdConfigGet{common: c.common, config: c}
	cmd.AddCommand(configGetCmd.Command())

	configDeleteCmd := &cmdConfigDelete{common: c.common, config: c}
	cmd.AddCommand(configDeleteCmd.Command())

	return cmd
}
