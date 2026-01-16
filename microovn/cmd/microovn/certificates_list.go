package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/ovn/paths"
	"github.com/spf13/cobra"
)

type cmdCertificatesList struct {
	common       *CmdControl
	certificates *cmdCertificates
	FormatFlag   string
}

// caCertInfo is structure that holds path to the CA certificate and
// information about whether microOVN will automatically renew it when
// it nears it's expiration.
type caCertInfo struct {
	Cert      string `json:"cert"`
	AutoRenew bool   `json:"auto_renew"`
}

// certBundle is structure for holding path to certificate and related private key
type certBundle struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

// ovnCertificatePaths is structure that holds paths to all certificates used by OVN
type ovnCertificatePaths struct {
	Ca      *caCertInfo `json:"ca"`
	Nb      *certBundle `json:"ovnnb"`
	Sb      *certBundle `json:"ovnsb"`
	Northd  *certBundle `json:"ovn-northd"`
	Chassis *certBundle `json:"ovn-controller"`
	Client  *certBundle `json:"client"`
}

var outputFormats = []string{"text", "json"}

// Command method returns definition for "microovn certificates list" subcommand
func (c *cmdCertificatesList) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List certificates and private keys currently used by OVN services",
		RunE:  c.Run,
	}

	allowedFormats := strings.Join(outputFormats, ", ")
	cmd.Flags().StringVarP(
		&c.FormatFlag,
		"format",
		"f",
		"text",
		fmt.Sprintf("Output format selector. (Allowed formats: %s)", allowedFormats),
	)
	return cmd
}

// Run method is an implementation of "microovn certificates list" subcommand
func (c *cmdCertificatesList) Run(cmd *cobra.Command, _ []string) error {
	// Get name of the local node
	localHostname, err := os.Hostname()
	if err != nil {
		return err
	}

	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	var expectedCertificates ovnCertificatePaths
	caInfo, err := client.GetCaInfo(context.Background(), cli)
	if err != nil {
		return err
	}
	if caInfo.Error != "" {
		return fmt.Errorf("%s", caInfo.Error)
	}
	expectedCertificates.Ca = &caCertInfo{
		Cert:      paths.PkiCaCertFile(),
		AutoRenew: caInfo.AutoRenew,
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

		if srv.Service == types.SrvCentral {
			nbCert, nbKey := paths.PkiOvnNbCertFiles()
			sbCert, sbKey := paths.PkiOvnSbCertFiles()
			northdCert, northdKey := paths.PkiOvnNorthdCertFiles()

			expectedCertificates.Nb = &certBundle{nbCert, nbKey}
			expectedCertificates.Sb = &certBundle{sbCert, sbKey}
			expectedCertificates.Northd = &certBundle{northdCert, northdKey}
		}

		if srv.Service == types.SrvChassis {
			ctlCert, ctlKey := paths.PkiOvnControllerCertFiles()
			expectedCertificates.Chassis = &certBundle{ctlCert, ctlKey}
		}
		clientCert, clientKey := paths.PkiClientCertFiles()
		expectedCertificates.Client = &certBundle{clientCert, clientKey}
	}

	outputFormat := cmd.Flag("format").Value.String()
	switch outputFormat {
	case "text":
		printOvnCertStatus(&expectedCertificates)
	case "json":
		jsonString, err := json.Marshal(expectedCertificates)
		if err != nil {
			return err
		}
		fmt.Printf("%s", string(jsonString))
	default:
		return fmt.Errorf("unknown output format specified: %s", outputFormat)
	}
	return nil
}

// printOvnCertStatus prints overall status of certificate bundles contained in
// "certificates" argument
func printOvnCertStatus(certificates *ovnCertificatePaths) {
	fmt.Println("[OVN CA]")
	if certificates.Ca.Cert == "" {
		fmt.Println("Error: missing")
	} else {
		printFileStatus(certificates.Ca.Cert)
		fmt.Printf("Auto-renew: %t\n", certificates.Ca.AutoRenew)
	}

	fmt.Println("\n[OVN Northbound Service]")
	printCertBundleStatus(certificates.Nb)

	fmt.Println("\n[OVN Southbound Service]")
	printCertBundleStatus(certificates.Sb)

	fmt.Println("\n[OVN Northd Service]")
	printCertBundleStatus(certificates.Northd)

	fmt.Println("\n[OVN Chassis Service]")
	printCertBundleStatus(certificates.Chassis)

	fmt.Println("\n[Client]")
	printCertBundleStatus(certificates.Client)
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
	fmt.Printf("%s (%s)\n", filePath, certStatus)
}
