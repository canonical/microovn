// Package microovn provides the main client tool.
package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	cli "github.com/canonical/lxd/shared/cmd"

	"github.com/canonical/microovn/microovn/version"
)

// CmdControl has functions that are common to the microctl commands.
// command line tools.
type CmdControl struct {
	FlagHelp       bool
	FlagVersion    bool
	FlagLogDebug   bool
	FlagLogVerbose bool
	FlagStateDir   string

	asker cli.Asker
}

func main() {
	// common flags.
	commonCmd := CmdControl{asker: cli.NewAsker(bufio.NewReader(os.Stdin), nil)}

	app := &cobra.Command{
		Use:               "microovn",
		Short:             "Command for managing the MicroOVN deployment",
		Version:           version.MicroOvnVersion,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	app.PersistentFlags().StringVar(&commonCmd.FlagStateDir, "state-dir", "", "Path to store state information"+"``")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagHelp, "help", "h", false, "Print help")
	app.PersistentFlags().BoolVar(&commonCmd.FlagVersion, "version", false, "Print version number")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagLogDebug, "debug", "d", false, "Show all debug messages")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagLogVerbose, "verbose", "v", false, "Show all information messages")

	app.SetVersionTemplate("{{.Version}}\n")
	app.Version = fmt.Sprintf("microovn: %s\novn: %s\nopenvswitch: %s",
		version.MicroOvnVersion, version.OvnVersion,
		version.OvsVersion)

	// Top-level.
	var cmdInit = cmdInit{common: &commonCmd}
	app.AddCommand(cmdInit.Command())

	var cmdStatus = cmdStatus{common: &commonCmd}
	app.AddCommand(cmdStatus.Command())

	var cmdDisable = cmdDisable{common: &commonCmd}
	app.AddCommand(cmdDisable.Command())

	var cmdEnable = cmdEnable{common: &commonCmd}
	app.AddCommand(cmdEnable.Command())

	// Nested.
	var cmdCluster = cmdCluster{common: &commonCmd}
	app.AddCommand(cmdCluster.Command())

	var cmdCertificates = cmdCertificates{common: &commonCmd}
	app.AddCommand(cmdCertificates.Command())

	app.InitDefaultHelpCmd()

	err := app.Execute()
	if err != nil {
		os.Exit(1)
	}
}
