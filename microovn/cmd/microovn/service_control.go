package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/client"
)

type cmdDisable struct {
	common                  *CmdControl
	allowDisableLastCentral bool
}

func (c *cmdDisable) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: "disable <SERVICE>",
		Short: fmt.Sprintf(
			"Disable selected service on the local node. (Valid service names: %s)",
			strings.Join(types.ServiceNames, ", "),
		),
		ValidArgs: types.ServiceNames,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE:      c.Run,
	}

	cmd.Flags().BoolVar(&c.allowDisableLastCentral, "allow-disable-last-central", false, "Allow disabling the last node of the central service. WARNING: If the last central service is disabled, OVN Northbound and Southbound databases will be removed!")

	return cmd
}

func (c *cmdDisable) Run(_ *cobra.Command, args []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	targetService := args[0]
	ws, regenEnv, err := client.DisableService(context.Background(), cli, targetService, c.allowDisableLastCentral)
	if err != nil {
		return err
	}
	fmt.Printf("Service %s disabled\n", targetService)
	ws.PrettyPrint(c.common.FlagLogVerbose)
	if c.common.FlagLogVerbose {
		regenEnv.PrettyPrint()
	}
	return nil
}

type cmdEnable struct {
	common      *CmdControl
	extraConfig []string
}

func (c *cmdEnable) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: "enable <SERVICE>",
		Short: fmt.Sprintf(
			"Enable selected service on the local node. (Valid service names: %s)",
			strings.Join(types.ServiceNames, ", "),
		),
		ValidArgs: types.ServiceNames,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE:      c.Run,
	}
	cmd.Flags().StringArrayVar(
		&c.extraConfig,
		"config",
		[]string{},
		"Additional configuration options for enabling service",
	)
	return cmd
}

func (c *cmdEnable) Run(_ *cobra.Command, args []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	targetService := args[0]
	extraConfig, err := c.parseExtraConfig(targetService)
	if err != nil {
		return err
	}

	ws, regenEnv, err := client.EnableService(context.Background(), cli, targetService, &extraConfig)

	if err != nil {
		return err
	}
	fmt.Printf("Service %s enabled\n", targetService)
	ws.PrettyPrint(c.common.FlagLogVerbose)
	if c.common.FlagLogVerbose {
		regenEnv.PrettyPrint()
	}
	return nil
}

// parseExtraConfig parses extra arguments passed to the cmdEnable in form of "--config key=value". Based
// on the service that's being enabled, it the initializes appropriate extra config structure from these values.
func (c *cmdEnable) parseExtraConfig(targetService types.SrvName) (types.ExtraServiceConfig, error) {
	extraConfig := types.ExtraServiceConfig{}
	rawConfig := map[string]string{}

	for _, configString := range c.extraConfig {
		key, value, found := strings.Cut(configString, "=")
		if !found {
			err := fmt.Errorf("configuration '%s' does not conform to the 'key=value' format", configString)
			return extraConfig, err
		}
		_, exists := rawConfig[key]
		if exists {
			err := fmt.Errorf("configuration '%s' already set", key)
			return extraConfig, err
		}
		rawConfig[key] = value
	}

	if len(rawConfig) == 0 {
		return extraConfig, nil
	}

	if targetService == types.SrvBgp {
		bgpConfig := types.ExtraBgpConfig{}
		err := bgpConfig.FromMap(rawConfig)
		if err != nil {
			return extraConfig, err
		}
		extraConfig.BgpConfig = &bgpConfig
	} else {
		return extraConfig, fmt.Errorf("service '%s' does not accpet extra config", targetService)
	}
	return extraConfig, nil
}
