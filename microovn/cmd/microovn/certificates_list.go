package main

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
	ExpDate   string `json:"expiration_date"`
}

// certBundle is structure for holding path to certificate and related private key
// as well as the cert's expiration date
type certBundle struct {
	Cert    string `json:"cert"`
	Key     string `json:"key"`
	ExpDate string `json:"expiration_date"`
}

type certProvider interface {
	CertPath() string
}

func (bundle *certBundle) CertPath() string {
	return bundle.Cert
}

func (caInfo *caCertInfo) CertPath() string {
	return caInfo.Cert
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

	caInfo, err := client.GetCaInfo(context.Background(), cli)
	if err != nil {
		return err
	}
	if caInfo.Error != "" {
		return fmt.Errorf("%s", caInfo.Error)
	}

	// Get list of all services in microovn
	services, err := client.GetServices(context.Background(), cli)
	if err != nil {
		return err
	}

	var expectedCertificates ovnCertificatePaths
	err = populateExpectedCertificates(&expectedCertificates, services, caInfo, localHostname)
	if err != nil {
		return err
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

func populateExpectedCertificates(expectedCertificates *ovnCertificatePaths, services types.Services, caInfo types.CaInfo, localHostname string) error {
	caCert := paths.PkiCaCertFile()
	caExpDate, _, err := certExpDate(caCert)
	if err != nil {
		return err
	}
	expectedCertificates.Ca = &caCertInfo{
		Cert:      caCert,
		AutoRenew: caInfo.AutoRenew,
		ExpDate:   caExpDate.String(),
	}
	// Gather paths to all certificates that should be running on local host
	for _, srv := range services {
		// Skip service that do not run on this member
		if srv.Location != localHostname {
			continue
		}

		if srv.Service == types.SrvCentral {
			nbCert, nbKey := paths.PkiOvnNbCertFiles()
			nbCertExpDate, _, _ := certExpDate(nbCert)
			sbCert, sbKey := paths.PkiOvnSbCertFiles()
			sbCertExpDate, _, _ := certExpDate(sbCert)
			northdCert, northdKey := paths.PkiOvnNorthdCertFiles()
			northdCertExpDate, _, _ := certExpDate(northdCert)

			expectedCertificates.Nb = &certBundle{nbCert, nbKey, nbCertExpDate.String()}
			expectedCertificates.Sb = &certBundle{sbCert, sbKey, sbCertExpDate.String()}
			expectedCertificates.Northd = &certBundle{northdCert, northdKey, northdCertExpDate.String()}
		}

		if srv.Service == types.SrvChassis {
			ctlCert, ctlKey := paths.PkiOvnControllerCertFiles()
			ctlCertExpDate, _, _ := certExpDate(ctlCert)
			expectedCertificates.Chassis = &certBundle{ctlCert, ctlKey, ctlCertExpDate.String()}
		}
		clientCert, clientKey := paths.PkiClientCertFiles()
		clientCertExpDate, _, _ := certExpDate(clientCert)
		expectedCertificates.Client = &certBundle{clientCert, clientKey, clientCertExpDate.String()}
	}
	return nil
}

// printOvnCertStatus prints overall status of certificate bundles contained in
// "certificates" argument
func printOvnCertStatus(certificates *ovnCertificatePaths) {

	fmt.Println("[OVN CA]")
	printCert(certificates.Ca)

	fmt.Println("\n[OVN Northbound Database]")
	printCert(certificates.Nb)

	fmt.Println("\n[OVN Southbound Database]")
	printCert(certificates.Sb)

	fmt.Println("\n[OVN Northd Service]")
	printCert(certificates.Northd)

	fmt.Println("\n[OVN Chassis Service]")
	printCert(certificates.Chassis)

	fmt.Println("\n[Client]")
	printCert(certificates.Client)
}

// printCert prints status of individual files in certificate bundle or status of CA
type printCertInterface interface {
	printCertStatus()
}

func printCert(p printCertInterface) {
	p.printCertStatus()
}

// printCertStatus prints status of individual files in certificate bundle
func (bundle *certBundle) printCertStatus() {
	if bundle == nil {
		fmt.Println("Not present.")
	} else {
		printFileStatus(bundle.Cert)
		printCertExpDate(bundle.ExpDate)
		printFileStatus(bundle.Key)
	}
}

// printCertStatus prints status of CA certificate
func (caInfo *caCertInfo) printCertStatus() {
	if caInfo == nil {
		fmt.Println("Not present.")
	} else {
		printFileStatus(caInfo.Cert)
		printCertExpDate(caInfo.ExpDate)
		fmt.Printf("Auto-renew: %t\n", caInfo.AutoRenew)
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

func printCertExpDate(expDate string) {
	fmt.Printf("expiration date: %s\n", expDate)
}

// certExpDate returns the expiration date of public certificates, (NotAfter, NotBefore)
func certExpDate(filePath string) (time.Time, time.Time, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	certData, _ := pem.Decode(data)
	if certData == nil {
		return time.Time{}, time.Time{}, errors.New("failed to decode certificate's PEM data")
	}
	cert, err := x509.ParseCertificate(certData.Bytes)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return cert.NotAfter, cert.NotBefore, nil
}

// certIsExpired returns true if certificate has expired, else false
func certIsExpired(filePath string) (bool, error) {
	certExpDate, certStartDate, err := certExpDate(filePath)
	if err == nil {
		return time.Now().After(certExpDate) || time.Now().Before(certStartDate), nil
	}
	return true, err
}
