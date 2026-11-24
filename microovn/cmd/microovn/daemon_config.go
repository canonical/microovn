package main

import (
	"github.com/spf13/cobra"
)

const (
	daemonConfigKeyName          = "name"
	daemonConfigKeyAddress       = "address"
	daemonConfigKeyServers       = "servers"
	daemonConfigKeyFailureDomain = "failure-domain"
)

func isKnownDaemonConfigKey(key string) bool {
	switch key {
	case daemonConfigKeyName, daemonConfigKeyAddress, daemonConfigKeyServers, daemonConfigKeyFailureDomain:
		return true
	default:
		return false
	}
}

type cmdDaemonConfig struct {
	common *CmdControl
}

// Command returns definition for "microovn daemon-config" subcommand
func (c *cmdDaemonConfig) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon-config",
		Short: "Manage the local MicroOVN daemon configuration",
	}

	daemonConfigGetCmd := &cmdDaemonConfigGet{common: c.common, daemonConfig: c}
	cmd.AddCommand(daemonConfigGetCmd.Command())

	daemonConfigSetCmd := &cmdDaemonConfigSet{common: c.common, daemonConfig: c}
	cmd.AddCommand(daemonConfigSetCmd.Command())

	daemonConfigListCmd := &cmdDaemonConfigList{common: c.common, daemonConfig: c}
	cmd.AddCommand(daemonConfigListCmd.Command())

	return cmd
}
