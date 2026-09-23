package main

import (
	"errors"
	"fmt"

	"github.com/canonical/lxd/shared/i18n"
	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/canonical/microovn/microovn/inspect"
	"github.com/spf13/cobra"
)

type outputFormat string

const (
	formatText outputFormat = "text"
	formatJSON outputFormat = "json"
)

type cmdInspect struct {
	common           *CmdControl
	flagVerbose      bool
	flagOutputFormat outputFormat
}

func (f outputFormat) String() string {
	return string(f)
}

func (f *outputFormat) Set(v string) error {
	switch outputFormat(v) {
	case formatJSON, formatText:
		*f = outputFormat(v)
		return nil
	default:
		return fmt.Errorf(i18n.G("Invalid format %q"), v)
	}
}

func (f *outputFormat) Type() string {
	return "string"
}

func (c *cmdInspect) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect the MicroOVN deployment",
		RunE:  c.Run,
	}

	c.flagOutputFormat = formatText

	cmd.Flags().BoolVarP(&c.flagVerbose, "verbose", "v", false, "Show full inspection results")
	cmd.Flags().VarP(&c.flagOutputFormat, "format", "f", i18n.G("Format (text|json)"))
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		c.common.ExitCode = 2
		return err
	})

	return cmd
}

func (c *cmdInspect) Run(cmd *cobra.Command, args []string) error {
	c.common.ExitCode = 2

	if len(args) != 0 {
		return errors.New(i18n.G("inspect accepts no positional arguments"))
	}
	if c.flagVerbose && c.flagOutputFormat == formatJSON {
		return errors.New(i18n.G("--verbose and --format=json cannot be used together"))
	}

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	report, err := inspect.Orchestrate(cmd.Context(), inspect.Dependencies{
		Cluster: m,
		Client:  cli,
		Checks:  inspect.DefaultChecks(),
	})
	if err != nil {
		return err
	}

	if c.flagOutputFormat == formatJSON {
		err = inspect.RenderJSON(cmd.OutOrStdout(), report)
	} else {
		err = inspect.RenderText(cmd.OutOrStdout(), report, c.flagVerbose)
	}
	if err != nil {
		return err
	}

	c.common.ExitCode = inspect.ExitCode(report)
	return nil
}
