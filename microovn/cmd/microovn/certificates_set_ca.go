package main

import (
	"bytes"
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/canonical/microcluster/v3/microcluster"
	"github.com/spf13/cobra"

	"github.com/canonical/microovn/microovn/client"
)

type cmdCertificatesSetCA struct {
	common       *CmdControl
	certificates *cmdCertificates
	certPath     string
	keyPath      string
	combined     bool
}

// Command method returns definition for "microovn certificates set-ca" subcommand
func (c *cmdCertificatesSetCA) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-ca",
		Short: "Set a new CA certificate and reissue all service certificates.",
		RunE:  c.Run,
	}

	cmd.Flags().StringVar(&c.certPath, "cert", "", "Path to the CA certificate file")
	cmd.Flags().StringVar(&c.keyPath, "key", "", "Path to the CA private key file")
	cmd.Flags().BoolVar(&c.combined, "combined", false, "CA certificate and CA private key are being fed in via stdin")

	return cmd
}

func extractCertsAndKey(data []byte) ([]byte, []byte, error) {
	var certsBuf bytes.Buffer
	var keyBuf bytes.Buffer

	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}

		switch block.Type {
		case "CERTIFICATE":
			err := pem.Encode(&certsBuf, block)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to encode certificate block: %w", err)
			}
		case "RSA PRIVATE KEY", "PRIVATE KEY", "EC PRIVATE KEY":
			// Only keep the first private key
			if keyBuf.Len() == 0 {
				err := pem.Encode(&keyBuf, block)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to encode key block: %w", err)
				}
			}
		default:
			// Ignore other block types
		}
	}

	if certsBuf.Len() == 0 {
		return nil, nil, fmt.Errorf("no certificates found")
	}
	if keyBuf.Len() == 0 {
		return nil, nil, fmt.Errorf("no private key found")
	}

	return certsBuf.Bytes(), keyBuf.Bytes(), nil
}

// Run method implements the functionality of "microovn certificates set-ca" command. It reads the provided
// certificate and key files, then requests local MicroOVN service to use them as CA and to reissue all service
// certificates on every node.
func (c *cmdCertificatesSetCA) Run(_ *cobra.Command, _ []string) error {
	var certData []byte
	var keyData []byte
	var err error

	certPathExists := c.certPath != ""
	keyPathExists := c.keyPath != ""
	if c.combined == (keyPathExists || certPathExists) {
		return errors.New("you must use --combined xor (--key and --cert)")
	}

	if keyPathExists != certPathExists {
		return errors.New("you must use either both or none of --key and --cert")
	}

	if c.combined {
		stdinData, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		certData, keyData, err = extractCertsAndKey(stdinData)

		if err != nil {
			return fmt.Errorf("failed to parse certificates and keys from stdin: %w", err)
		}
	} else {
		certData, err = os.ReadFile(c.certPath)
		if err != nil {
			return fmt.Errorf("failed to read certificate file: %w", err)
		}

		keyData, err = os.ReadFile(c.keyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key file: %w", err)
		}
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
