package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/spf13/cobra"
)

type cmdDaemonConfigGet struct {
	common       *CmdControl
	daemonConfig *cmdDaemonConfig
}

// Command returns definition for "microovn daemon-config get" subcommand
func (c *cmdDaemonConfigGet) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <KEY>",
		Short: "Get a value from the local daemon configuration",
		Args:  cobra.ExactArgs(1),
		RunE:  c.Run,
	}
	return cmd
}

// Run method is an implementation of the "microovn daemon-config get" subcommand
func (c *cmdDaemonConfigGet) Run(_ *cobra.Command, args []string) error {
	key := args[0]

	if !isKnownDaemonConfigKey(key) {
		return fmt.Errorf("daemon configuration key '%s' is not recognized", key)
	}

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	config, err := m.GetDaemonConfig(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get daemon configuration key '%s': %w", key, err)
	}

	switch key {
	case daemonConfigKeyName:
		fmt.Println(config.Name)
	case daemonConfigKeyAddress:
		fmt.Println(config.Address.String())
	case daemonConfigKeyServers:
		if len(config.Servers) == 0 {
			fmt.Println("{}")
			return nil
		}
		out, err := json.Marshal(config.Servers)
		if err != nil {
			return fmt.Errorf("failed to encode daemon servers: %w", err)
		}
		fmt.Println(string(out))
	case daemonConfigKeyFailureDomain:
		fmt.Println(config.FailureDomain)
	default:
		return fmt.Errorf("daemon configuration key '%s' is not recognized", key)
	}

	return nil
}
