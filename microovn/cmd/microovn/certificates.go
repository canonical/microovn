package main

import (
	"github.com/spf13/cobra"
)

type cmdCertificates struct {
	common *CmdControl
}

// Command returns definition for "microovn certificates" subcommand
func (c *cmdCertificates) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "certificates",
		Short: "Manage OVN certificates",
	}

	certificatesListCmd := cmdCertificatesList{common: c.common, certificates: c}
	cmd.AddCommand(certificatesListCmd.Command())

	certificatesReissueCmd := cmdCertificatesReissue{common: c.common, certificates: c}
	cmd.AddCommand(certificatesReissueCmd.Command())

	certificatesRegenerateCa := cmdCertificatesRegenerateCa{common: c.common, certificates: c}
	cmd.AddCommand(certificatesRegenerateCa.Command())

	return cmd
}
