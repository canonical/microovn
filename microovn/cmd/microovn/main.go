// Package microovn provides the main client tool.
package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/version"
)

// CmdControl has functions that are common to the microctl commands.
// command line tools.
type CmdControl struct {
	cmd *cobra.Command

	FlagHelp       bool
	FlagVersion    bool
	FlagLogDebug   bool
	FlagLogVerbose bool
	FlagStateDir   string
}

func main() {
	// common flags.
	commonCmd := CmdControl{}

	app := &cobra.Command{
		Use:               "microovn",
		Short:             "Command for managing the MicroOVN deployment",
		Version:           version.Version,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	app.PersistentFlags().StringVar(&commonCmd.FlagStateDir, "state-dir", "", "Path to store state information"+"``")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagHelp, "help", "h", false, "Print help")
	app.PersistentFlags().BoolVar(&commonCmd.FlagVersion, "version", false, "Print version number")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagLogDebug, "debug", "d", false, "Show all debug messages")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagLogVerbose, "verbose", "v", false, "Show all information messages")

	app.SetVersionTemplate("{{.Version}}\n")

	// Top-level.
	var cmdInit = cmdInit{common: &commonCmd}
	app.AddCommand(cmdInit.Command())

	var cmdStatus = cmdStatus{common: &commonCmd}
	app.AddCommand(cmdStatus.Command())

	// Nested.
	var cmdCluster = cmdCluster{common: &commonCmd}
	app.AddCommand(cmdCluster.Command())

	app.InitDefaultHelpCmd()

	err := app.Execute()
	if err != nil {
		os.Exit(1)
	}
}
