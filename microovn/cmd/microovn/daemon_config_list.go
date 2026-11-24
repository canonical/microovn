package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type cmdDaemonConfigList struct {
	common       *CmdControl
	daemonConfig *cmdDaemonConfig
}

// Command returns definition for "microovn daemon-config list" subcommand
func (c *cmdDaemonConfigList) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List daemon configuration keys",
		RunE:  c.Run,
	}
	return cmd
}

// Run method is an implementation of the "microovn daemon-config list" subcommand
func (c *cmdDaemonConfigList) Run(_ *cobra.Command, _ []string) error {
	fmt.Println("Available options:")
	fmt.Println(daemonConfigKeyName)
	fmt.Println(daemonConfigKeyAddress)
	fmt.Println(daemonConfigKeyServers)
	fmt.Println(daemonConfigKeyFailureDomain)
	return nil
}
