package main

import (
	"context"
	"fmt"
	"os"

	"github.com/canonical/microcluster/microcluster"
	"github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/ovn/paths"
	"github.com/spf13/cobra"
)

type cmdCertificatesList struct {
	common       *CmdControl
	certificates *cmdCertificates
}

// certBundle is structure for holding path to certificate and related private key
type certBundle struct {
	Cert string
	Key  string
}

// ovnCertificatePaths is structure that holds paths to all certificates used by OVN
type ovnCertificatePaths struct {
	ca      string
	nb      *certBundle
	sb      *certBundle
	northd  *certBundle
	chassis *certBundle
}

// Command method returns definition for "microovn certificates list" subcommand
func (c *cmdCertificatesList) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List certificates and private keys currently used by OVN services",
		RunE:  c.Run,
	}

	return cmd
}

// Run method is an implementation of "microovn certificates list" subcommand
func (c *cmdCertificatesList) Run(_ *cobra.Command, _ []string) error {
	var expectedCertificates ovnCertificatePaths
	expectedCertificates.ca = paths.PkiCaCertFile()

	// Get name of the local node
	localHostname, err := os.Hostname()
	if err != nil {
		return err
	}

	m, err := microcluster.App(context.Background(), microcluster.Args{StateDir: c.common.FlagStateDir, Verbose: c.common.FlagLogVerbose, Debug: c.common.FlagLogDebug})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	// Get list of all services in microovn
	services, err := client.GetServices(context.Background(), cli)
	if err != nil {
		return err
	}

	// Gather paths to all certificates that should be running on local host
	for _, srv := range services {
		// Skip service that do not run on this member
		if srv.Location != localHostname {
			continue
		}

		if srv.Service == "central" {
			nbCert, nbKey := paths.PkiOvnNbCertFiles()
			sbCert, sbKey := paths.PkiOvnSbCertFiles()
			northdCert, northdKey := paths.PkiOvnNorthdCertFiles()

			expectedCertificates.nb = &certBundle{nbCert, nbKey}
			expectedCertificates.sb = &certBundle{sbCert, sbKey}
			expectedCertificates.northd = &certBundle{northdCert, northdKey}
		}

		if srv.Service == "switch" {
			ctlCert, ctlKey := paths.PkiOvnControllerCertFiles()
			expectedCertificates.chassis = &certBundle{ctlCert, ctlKey}
		}
	}

	printOvnCertStatus(&expectedCertificates)
	return nil
}

// printOvnCertStatus prints overall status of certificate bundles contained in
// "certificates" argument
func printOvnCertStatus(certificates *ovnCertificatePaths) {
	fmt.Println("[OVN CA]")
	if certificates.ca == "" {
		fmt.Println("Error: missing")
	} else {
		printFileStatus(certificates.ca)
	}

	fmt.Println("\n[OVN Northbound Service]")
	printCertBundleStatus(certificates.nb)

	fmt.Println("\n[OVN Southbound Service]")
	printCertBundleStatus(certificates.sb)

	fmt.Println("\n[OVN Northd Service]")
	printCertBundleStatus(certificates.northd)

	fmt.Println("\n[OVN Chassis Service]")
	printCertBundleStatus(certificates.chassis)
}

// printCertBundleStatus prints status of individual files in certificate bundle
func printCertBundleStatus(bundle *certBundle) {
	if bundle == nil {
		fmt.Println("Not present.")
	} else {
		printFileStatus(bundle.Cert)
		printFileStatus(bundle.Key)
	}
}

// printFileStatus prints supplied file path with status base on whether the file exists or not
func printFileStatus(filePath string) {
	_, err := os.Stat(filePath)
	var certStatus string

	if err != nil {
		certStatus = "Error: Missing file"
	} else {
		certStatus = "OK: Present"
	}
	fmt.Println(fmt.Sprintf("%s (%s)", filePath, certStatus))
}
