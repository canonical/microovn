package main

import (
	"context"
	"fmt"
	"os"

	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/client"
)

type cmdCertificatesSetCA struct {
	common       *CmdControl
	certificates *cmdCertificates
	certPath     string
	keyPath      string
}

// Command method returns definition for "microovn certificates set-ca" subcommand
func (c *cmdCertificatesSetCA) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-ca",
		Short: "Set a new CA certificate and reissue all service certificates.",
		RunE:  c.Run,
	}

	cmd.Flags().StringVar(&c.certPath, "cert", "", "Path to the CA certificate file (required)")
	cmd.Flags().StringVar(&c.keyPath, "key", "", "Path to the CA private key file (required)")
	_ = cmd.MarkFlagRequired("cert")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}

// Run method implements the functionality of "microovn certificates set-ca" command. It reads the provided
// certificate and key files, then requests local MicroOVN service to use them as CA and to reissue all service
// certificates on every node.
func (c *cmdCertificatesSetCA) Run(_ *cobra.Command, _ []string) error {
	certData, err := os.ReadFile(c.certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyData, err := os.ReadFile(c.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	response, err := client.SetCA(context.Background(), cli, string(certData), string(keyData))
	if err != nil {
		return fmt.Errorf("command failed: %s", err)
	}

	response.PrettyPrint()
	return nil
}
