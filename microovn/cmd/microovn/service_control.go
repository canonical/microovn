package main

import (
	"context"
	"fmt"

	"github.com/canonical/microcluster/microcluster"
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
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir, Verbose: c.common.FlagLogVerbose, Debug: c.common.FlagLogDebug})
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
	err = client.DisableService(context.Background(), cli, targetService)

	if err != nil {
		return err
	}
	fmt.Printf("Service %s disabled\n", targetService)
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
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir, Verbose: c.common.FlagLogVerbose, Debug: c.common.FlagLogDebug})
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
	err = client.EnableService(context.Background(), cli, targetService)

	if err != nil {
		return err
	}
	fmt.Printf("Service %s enabled\n", targetService)
	return nil
}
