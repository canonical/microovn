// Package microovnd provides the daemon.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/canonical/microcluster/v3/state"
	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/api"
	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/ovn"
	"github.com/canonical/microovn/microovn/version"
)

// Debug indicates whether to log debug messages or not.
var Debug bool

// Verbose indicates verbosity.
var Verbose bool

type cmdGlobal struct {
	cmd *cobra.Command //nolint:structcheck,unused // FIXME: Remove the nolint flag when this is in use.

	flagHelp    bool
	flagVersion bool

	flagLogDebug   bool
	flagLogVerbose bool
}

func (c *cmdGlobal) Run(_ *cobra.Command, _ []string) error {
	Debug = c.flagLogDebug
	Verbose = c.flagLogVerbose

	return logger.InitLogger("", "", c.flagLogVerbose, c.flagLogDebug, nil)
}

type cmdDaemon struct {
	global *cmdGlobal

	flagStateDir string
}

func (c *cmdDaemon) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "microd",
		Short:   "Example daemon for MicroCluster - This will start a daemon with a running control socket and no database",
		Version: version.MicroOvnVersion,
	}

	cmd.RunE = c.Run

	return cmd
}

func (c *cmdDaemon) Run(_ *cobra.Command, _ []string) error {
	logLevel := slog.LevelInfo
	if c.global.flagLogDebug {
		logLevel = slog.LevelDebug
	}

	// Create our own logging handler to modify the log level and output.
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	m, err := microcluster.App(microcluster.Args{StateDir: c.flagStateDir, LogHandler: logHandler})
	if err != nil {
		return err
	}

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := &state.Hooks{}
	h.PostBootstrap = ovn.Bootstrap
	h.PreJoin = ovn.Join
	h.OnNewMember = func(ctx context.Context, s state.State, newMember types.ClusterMemberLocal) error {
		ovn.Refresh(ctx, s)
		return nil
	}
	h.PreRemove = ovn.Leave
	h.PostRemove = func(ctx context.Context, s state.State, _ bool) error {
		ovn.Refresh(ctx, s)
		return nil
	}
	h.OnStart = ovn.Start

	daemonArgs := microcluster.DaemonArgs{
		Version:          version.MajorVersion(version.OvnVersion),
		ExtensionsSchema: database.SchemaExtensions,
		APIExtensions:    api.Extensions(),
		Hooks:            h,
		ExtensionServers: api.Server,
	}

	return m.Start(shutdownCtx, daemonArgs)
}

func main() {
	daemonCmd := cmdDaemon{global: &cmdGlobal{}}
	app := daemonCmd.Command()
	app.SilenceUsage = true
	app.CompletionOptions = cobra.CompletionOptions{DisableDefaultCmd: true}

	app.PersistentFlags().BoolVarP(&daemonCmd.global.flagHelp, "help", "h", false, "Print help")
	app.PersistentFlags().BoolVar(&daemonCmd.global.flagVersion, "version", false, "Print version number")
	app.PersistentFlags().BoolVarP(&daemonCmd.global.flagLogDebug, "debug", "d", false, "Show all debug messages")
	app.PersistentFlags().BoolVarP(&daemonCmd.global.flagLogVerbose, "verbose", "v", false, "Show all information messages")

	app.PersistentFlags().StringVar(&daemonCmd.flagStateDir, "state-dir", "", "Path to store state information"+"``")

	app.SetVersionTemplate("{{.Version}}\n")

	err := app.Execute()
	if err != nil {
		os.Exit(1)
	}
}
