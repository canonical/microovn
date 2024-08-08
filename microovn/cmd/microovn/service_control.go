package main

import (
	"context"
	"fmt"

	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/client"
)

type cmdDisable struct {
	common *CmdControl
}

func (c *cmdDisable) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <SERVICE>",
		Short: "disables a service",
		RunE:  c.Run,
	}

	return cmd
}

func (c *cmdDisable) Run(cmd *cobra.Command, args []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	targetService := args[0]
	ws, err := client.DisableService(context.Background(), cli, targetService)

	if err != nil {
		return err
	}
	fmt.Printf("Service %s disabled\n", targetService)
	ws.PrettyPrint(c.common.FlagLogVerbose)
	return nil
}

type cmdEnable struct {
	common *CmdControl
}

func (c *cmdEnable) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <SERVICE>",
		Short: "enables a service",
		RunE:  c.Run,
	}

	return cmd
}

func (c *cmdEnable) Run(cmd *cobra.Command, args []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	targetService := args[0]
	ws, err := client.EnableService(context.Background(), cli, targetService)

	if err != nil {
		return err
	}
	fmt.Printf("Service %s enabled\n", targetService)
	ws.PrettyPrint(c.common.FlagLogVerbose)
	return nil
}
