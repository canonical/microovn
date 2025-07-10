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
	common *CmdControl
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
	ws, regenEnv, err := client.EnableService(context.Background(), cli, targetService)

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
