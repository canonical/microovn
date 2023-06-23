package ovn

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/canonical/microcluster/state"
	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/ovn/paths"
)

const CACertRecordName = "ca_cert"                   // Key used to store CA certificate in config DB table
const CAKeyRecordName = "ca_key"                     // Key used to store CA private key in config DB table
const CACertValidity = 10 * 365 * 24 * time.Hour     // 10 years
const ServiceCertValidity = 2 * 365 * 24 * time.Hour // 2 years
const certFileMode = 0600

var MaxSerialNumber = new(big.Int).Lsh(big.NewInt(1), 128)

type CertificateType int

const (
	CertificateTypeCA CertificateType = iota
	CertificateTypeServer
	CertificateTypeClient
)

// issueCertificate function generates new "server", "client" or "CA" certificate based on the value passed
// to the "certType" argument. Argument "cn" is passed unchanged to the certificate's CN and "serviceName" is used in
// certificate's OU.  Certificates are valid, based on their type, for a period specified in CACertValidity
// and ServiceCertValidity.
//
// When generating certificate that is signed by a CA, "parent" argument must point to a parsed CA certificate and
// "signer" argument must point to CA's private key. On the other hand if you want to generate self-signed certificate,
// both "parent" and "signer" arguments must be empty (nil).
//
// This function returns PEM encoded certificate, private key and error (if any occurred).
func issueCertificate(cn string, serviceName string, certType CertificateType, parent *x509.Certificate, signer *ecdsa.PrivateKey) ([]byte, []byte, error) {
	var (
		isCa     bool
		keyUsage x509.KeyUsage
		signKey  *ecdsa.PrivateKey
		validTo  time.Time
	)
	// Generate certificate's private key
	// ECDSA with P-384 curve was selected to be in line with certificates used in LXD
	keyPair, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair for '%s' certificate: %w", serviceName, err)
	}

	// Generate certificate data and properties
	validFrom := time.Now().UTC()
	serialNumber, err := rand.Int(rand.Reader, MaxSerialNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to generate serial number: %w", err)
	}

	if certType == CertificateTypeCA {
		isCa = true
		keyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
		validTo = validFrom.Add(CACertValidity)
	} else if certType == CertificateTypeServer {
		isCa = false
		keyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageKeyAgreement | x509.KeyUsageContentCommitment
		validTo = validFrom.Add(ServiceCertValidity)
	} else if certType == CertificateTypeClient {
		isCa = false
		keyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyAgreement
		validTo = validFrom.Add(ServiceCertValidity)
	} else {
		return nil, nil, fmt.Errorf("failed ot issue certificate: unknown certificate type")
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         cn,
			Organization:       []string{"MicroOVN"},
			OrganizationalUnit: []string{serviceName},
		},

		NotBefore: validFrom,
		NotAfter:  validTo,

		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
		IsCA:                  isCa,
	}

	// If there's no parent, use certificate's own key to self-sign it.
	if parent == nil {
		parent = &template
		signKey = keyPair
	} else {
		signKey = signer
	}

	cert, err := x509.CreateCertificate(rand.Reader, &template, parent, &keyPair.PublicKey, signKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate for %s: %w", serviceName, err)
	}

	key, err := x509.MarshalECPrivateKey(keyPair)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: key})

	return certPEM, keyPEM, err
}

// GenerateNewCACertificate generates new CA certificate and private key and stores them in the shared MicroOVN
// database.
func GenerateNewCACertificate(s *state.State) error {
	cert, key, err := issueCertificate("MicroOVN CA", "MicroOVN CA", CertificateTypeCA, nil, nil)
	if err != nil {
		return err
	}

	caCert := database.ConfigItem{
		Key:   CACertRecordName,
		Value: string(cert),
	}
	caKey := database.ConfigItem{
		Key:   CAKeyRecordName,
		Value: string(key),
	}

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		// Upsert CA certificate
		certExists, _ := database.GetConfigItem(ctx, tx, CACertRecordName)
		if certExists == nil {
			_, err = database.CreateConfigItem(ctx, tx, caCert)
		} else {
			err = database.UpdateConfigItem(ctx, tx, CACertRecordName, caCert)
		}

		if err != nil {
			return fmt.Errorf("failed to store CA certificate in the database: %s", err)
		}

		// Upsert CA private key
		keyExists, _ := database.GetConfigItem(ctx, tx, CAKeyRecordName)
		if keyExists == nil {
			_, err = database.CreateConfigItem(ctx, tx, caKey)
		} else {
			err = database.UpdateConfigItem(ctx, tx, CAKeyRecordName, caKey)
		}

		if err != nil {
			return fmt.Errorf("failed to store CA private key in the database: %s", err)
		}

		return err
	})

	return err
}

// DumpCA copies CA certificate from shared database and stores it in pre-defined file on disk. File path
// to store CA certificate is defined in paths.PkiCaCertFile.
func DumpCA(s *state.State) error {
	var err error
	var CACertRecord *database.ConfigItem

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		CACertRecord, err = database.GetConfigItem(ctx, tx, CACertRecordName)
		if err != nil {
			return fmt.Errorf("failed to store CA certificate in the database: %s", err)
		}
		return err
	})

	certPath := paths.PkiCaCertFile()
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create file for CA certificate: %w", err)
	}
	defer certFile.Close()

	err = certFile.Chmod(certFileMode)
	if err != nil {
		return fmt.Errorf("unable to set permissions for CA certificate: %w", err)
	}

	_, err = certFile.WriteString(CACertRecord.Value)
	if err != nil {
		return fmt.Errorf("failed to write CA certificate into file %s: %w", certPath, err)
	}
	return nil
}

// getCA pulls PEM encoded CA certificate and private key from shared database and returns
// them as parsed objects x509.Certificate and ecdsa.PrivateKey (+ error if any occurred).
func getCA(s *state.State) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	var err error
	var CACertRecord *database.ConfigItem
	var CAKeyRecord *database.ConfigItem

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		CACertRecord, err = database.GetConfigItem(ctx, tx, CACertRecordName)
		if err != nil {
			return fmt.Errorf("failed to fetch CA certificate from database: %s", err)
		}

		CAKeyRecord, err = database.GetConfigItem(ctx, tx, CAKeyRecordName)
		if err != nil {
			return fmt.Errorf("failed to fetch CA private key from database: %s", err)
		}

		return err
	})

	if err != nil {
		return nil, nil, err
	}

	certData, _ := pem.Decode([]byte(CACertRecord.Value))
	if certData == nil {
		return nil, nil, errors.New("failed to decode CA certificate's PEM data")
	}

	keyData, _ := pem.Decode([]byte(CAKeyRecord.Value))
	if keyData == nil {
		return nil, nil, errors.New("failed to decode CA Private Key's PEM data")
	}

	caCert, err := x509.ParseCertificate(certData.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %s", err)
	}
	caKey, err := x509.ParseECPrivateKey(keyData.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA Private Key: %s", err)
	}

	return caCert, caKey, nil
}

// GenerateNewServiceCertificate creates new certificate, signs it with CA certificate stored in the shared database
// and writes resulting certificate and private key to files specified by certPath and keyPath arguments.
// String from serviceName argument will be inserted in certificate's OU and is meant to more easily distinguish
// between multiple certificates with same CN.
func GenerateNewServiceCertificate(s *state.State, serviceName string, certType CertificateType) error {
	certPath, keyPath, err := getServiceCertificatePaths(serviceName)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %s", err)
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create file for %s certificate: %w", serviceName, err)
	}
	defer certFile.Close()

	err = certFile.Chmod(certFileMode)
	if err != nil {
		return fmt.Errorf("unable to set permissions for %s certificate: %w", serviceName, err)
	}

	keyFile, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("failed to create file for %s private key: %w", serviceName, err)
	}
	defer keyFile.Close()

	err = keyFile.Chmod(certFileMode)
	if err != nil {
		return fmt.Errorf("unable to set permissions for %s private key: %w", serviceName, err)
	}

	caCert, caKey, err := getCA(s)
	if err != nil {
		return err
	}

	cert, key, err := issueCertificate(s.Name(), serviceName, certType, caCert, caKey)

	_, err = certFile.Write(cert)
	if err != nil {
		return fmt.Errorf("failed to write %s certificate into file %s: %w", serviceName, certPath, err)
	}

	_, err = keyFile.Write(key)
	if err != nil {
		return fmt.Errorf("failed to write %s private key into file %s: %w", serviceName, keyPath, err)
	}

	return nil
}

// getServiceCertificatePaths returns paths to certificate and private key based on service name
func getServiceCertificatePaths(service string) (string, string, error) {
	var (
		certPath string
		keyPath  string
		err      error
	)

	switch service {
	case "ovnnb":
		certPath, keyPath = paths.PkiOvnNbCertFiles()
	case "ovnsb":
		certPath, keyPath = paths.PkiOvnSbCertFiles()
	case "ovn-northd":
		certPath, keyPath = paths.PkiOvnNorthdCertFiles()
	case "ovn-controller":
		certPath, keyPath = paths.PkiOvnControllerCertFiles()
	default:
		certPath = ""
		keyPath = ""
		err = fmt.Errorf("unknown service '%s'. Can't generate certificate paths", service)
	}

	return certPath, keyPath, err
}
