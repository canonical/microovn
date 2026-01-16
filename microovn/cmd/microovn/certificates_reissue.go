package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/client"
)

var validCertificates = []string{
	"client",
	"ovnnb",
	"ovnsb",
	"ovn-controller",
	"ovn-northd",
	"all",
}

type cmdCertificatesReissue struct {
	common       *CmdControl
	certificates *cmdCertificates
}

// Command method returns definition for "microovn certificates reissue" subcommand
func (c *cmdCertificatesReissue) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: "reissue <SERVICE>",
		Short: fmt.Sprintf(
			"Reissue certificate for specified SERVICE on the local node. (Valid service names: %s)",
			strings.Join(validCertificates, ", "),
		),
		ValidArgs: validCertificates,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE:      c.Run,
	}

	return cmd
}

// Run method is an implementation of "microovn certificates reissue" subcommand. It requests local MicroOVN
// service to issue new certificate for selected OVN service.
func (c *cmdCertificatesReissue) Run(_ *cobra.Command, args []string) error {
	var response types.IssueCertificateResponse
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	targetService := args[0]

	if targetService == "all" {
		response, err = client.ReissueAllCertificate(context.Background(), cli)
	} else {
		response, err = client.ReissueCertificate(context.Background(), cli, targetService)
	}

	if err != nil {
		return fmt.Errorf("command failed: %s", err)
	}

	response.PrettyPrint()
	return nil
}
