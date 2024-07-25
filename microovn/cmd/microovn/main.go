// Package microovn provides the main client tool.
package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	cli "github.com/canonical/lxd/shared/cmd"
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

	asker cli.Asker
}

// MicroOvnVersion contains version of MicroOVN (set at build time)
var MicroOvnVersion string

// OvnVersion contains version of 'ovn' package used to build MicroOVN (set at build time)
var OvnVersion string

// OvsVersion contains version of 'openvswitch' package used to build MicroOVN (set at build time)
var OvsVersion string

func main() {
	// common flags.
	commonCmd := CmdControl{asker: cli.NewAsker(bufio.NewReader(os.Stdin))}

	app := &cobra.Command{
		Use:               "microovn",
		Short:             "Command for managing the MicroOVN deployment",
		Version:           MicroOvnVersion,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	app.PersistentFlags().StringVar(&commonCmd.FlagStateDir, "state-dir", "", "Path to store state information"+"``")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagHelp, "help", "h", false, "Print help")
	app.PersistentFlags().BoolVar(&commonCmd.FlagVersion, "version", false, "Print version number")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagLogDebug, "debug", "d", false, "Show all debug messages")
	app.PersistentFlags().BoolVarP(&commonCmd.FlagLogVerbose, "verbose", "v", false, "Show all information messages")

	app.SetVersionTemplate("{{.Version}}\n")
	app.Version = fmt.Sprintf("microovn: %s\novn: %s\nopenvswitch: %s", MicroOvnVersion, OvnVersion, OvsVersion)

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
