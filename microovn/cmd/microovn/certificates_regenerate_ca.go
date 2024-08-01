package main

import (
	"context"
	"fmt"

	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/client"
)

type cmdCertificatesRegenerateCa struct {
	common       *CmdControl
	certificates *cmdCertificates
}

// Command method returns definition for "microovn certificates regenerate-ca" subcommand
func (c *cmdCertificatesRegenerateCa) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regenerate-ca",
		Short: "Recreate new CA certificate and re-issue certificates for existing services across whole cluster.",
		RunE:  c.Run,
	}

	return cmd
}

// Run method is an implementation of "microovn certificates regenerate-ca" subcommand. It requests cluster
// to issue new CA certificate and re-issue all OVN service certificates across the whole cluster.
func (c *cmdCertificatesRegenerateCa) Run(_ *cobra.Command, _ []string) error {
	var response types.RegenerateCaResponse
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir, Verbose: c.common.FlagLogVerbose, Debug: c.common.FlagLogDebug})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	response, err = client.RegenerateCA(context.Background(), cli)

	if err != nil {
		return fmt.Errorf("command failed: %s", err)
	}

	response.PrettyPrint()
	return nil
}
