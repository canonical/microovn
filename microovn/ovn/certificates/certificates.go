// Package certificates is for exposing certificate generation functionality within
// ovn
package certificates

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/microcluster/v2/state"
	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/ovn/paths"
)

// CACertRecordName    - Key used to store CA certificate in config DB table.
const CACertRecordName = "ca_cert"

// CAKeyRecordName     - Key used to store CA private key in config DB table.
const CAKeyRecordName = "ca_key"

// CAAutoRenew 		   - Key used to store auto-renew flag in config DB table.
const CAAutoRenew = "ca_auto_renew"

// CaIsRenewable       -  value of CAAutoRenew signifying that the CA certificate
// is managed by MicroOVN and is eligible to be automatically renewed
const CaIsRenewable = "yes"

// CaIsNotRenewable    - value of CAAutoRenew signifying that the CA certificate
// is provided by the user and is NOT eligible to be automatically renewed
const CaIsNotRenewable = "no"

// CACertValidity      - 10 years.
const CACertValidity = 10 * 365 * 24 * time.Hour

// ServiceCertValidity -  2 years.
const ServiceCertValidity = 2 * 365 * 24 * time.Hour

const certFileMode = 0600

// MaxSerialNumber - Maximum serial number for generated certificates.
var MaxSerialNumber = new(big.Int).Lsh(big.NewInt(1), 128)

// CertificateType - Types of certificates.
type CertificateType int

const (
	// CertificateTypeCA     - A Certificate Authority certificate.
	CertificateTypeCA CertificateType = iota
	// CertificateTypeServer - A certificate suitable for a server.
	CertificateTypeServer
	// CertificateTypeClient - A certificate suitable for clients.
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
func issueCertificate(cn string, serviceName string, certType CertificateType, parent *x509.Certificate, signer any) ([]byte, []byte, error) {
	var (
		isCa     bool
		keyUsage x509.KeyUsage
		signKey  any
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
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	switch certType {
	case CertificateTypeCA:
		isCa = true
		keyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
		validTo = validFrom.Add(CACertValidity)
	case CertificateTypeServer:
		isCa = false
		keyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageKeyAgreement | x509.KeyUsageContentCommitment
		validTo = validFrom.Add(ServiceCertValidity)
	case CertificateTypeClient:
		isCa = false
		keyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyAgreement
		validTo = validFrom.Add(ServiceCertValidity)
	default:
		return nil, nil, fmt.Errorf("failed to issue certificate: unknown certificate type")
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

	key, err := x509.MarshalPKCS8PrivateKey(keyPair)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: key})

	return certPEM, keyPEM, err
}

// GenerateNewCACertificate generates new CA certificate and private key and stores them in the shared MicroOVN
// database.
func GenerateNewCACertificate(ctx context.Context, s state.State) error {
	cert, key, err := issueCertificate("MicroOVN CA", "MicroOVN CA", CertificateTypeCA, nil, nil)
	if err != nil {
		return err
	}

	return storeCA(ctx, s, string(cert), string(key), true)
}

// SetNewCACertificate verifies basic attributes of the provided certificate and private key and
// stores them in the shared MicroOVN database.
func SetNewCACertificate(ctx context.Context, s state.State, certPEM string, keyPEM string) error {
	certificates, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return fmt.Errorf("error parsing CA certificate: %w", err)
	}

	for _, parsedCert := range certificates.Certificate {
		x509Cert, err := x509.ParseCertificate(parsedCert)
		if err != nil {
			return fmt.Errorf("error parsing X509 certificate: %w", err)
		}

		if !x509Cert.IsCA {
			return fmt.Errorf("provided certificate is not a CA certificate")
		}

		if x509Cert.KeyUsage&x509.KeyUsageCertSign != x509.KeyUsageCertSign {
			return fmt.Errorf("provided certificate does not have the required keyCertSign KeyUsage")
		}
	}

	return storeCA(ctx, s, certPEM, keyPEM, false)
}

// storeCA saves the CA certificate and private key in a PEM format into the shared database. It also
// sets the config variable CAAutoRenew to "yes" or "no", based on the value of the autoRenew argument.
// Argument autoRenew controls whether MicroOVN will automatically re-generate new CA when the current one
// is nearing its expiration date. It should be set to 'true' if the CA is automatically generated by the MicroOVN,
// and to 'false' if the CA is managed externally (e.g. it's supplied by the user)
func storeCA(ctx context.Context, s state.State, certPEM string, keyPEM string, autoRenew bool) error {
	var err error

	caCert := database.ConfigItem{
		Key:   CACertRecordName,
		Value: certPEM,
	}
	caKey := database.ConfigItem{
		Key:   CAKeyRecordName,
		Value: keyPEM,
	}

	var autoRenewValue string
	if autoRenew {
		autoRenewValue = CaIsRenewable
	} else {
		autoRenewValue = CaIsNotRenewable
	}
	autoRenewFlag := database.ConfigItem{
		Key:   CAAutoRenew,
		Value: autoRenewValue,
	}

	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
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

		// Upsert auto-renew flag
		flagExists, _ := database.GetConfigItem(ctx, tx, CAAutoRenew)
		if flagExists == nil {
			_, err = database.CreateConfigItem(ctx, tx, autoRenewFlag)
		} else {
			err = database.UpdateConfigItem(ctx, tx, CAAutoRenew, autoRenewFlag)
		}

		if err != nil {
			return fmt.Errorf("failed to store CA auto-renew flag in the database: %s", err)
		}

		return err
	})

	return err
}

// DumpCA copies CA certificate from shared database and stores it in pre-defined file on disk. File path
// to store CA certificate is defined in paths.PkiCaCertFile.
func DumpCA(ctx context.Context, s state.State) error {
	var err error
	var CACertRecord *database.ConfigItem

	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		CACertRecord, err = database.GetConfigItem(ctx, tx, CACertRecordName)
		if err != nil {
			return fmt.Errorf("failed to get CA certificate from the database: %s", err)
		}
		return err
	})
	if err != nil {
		return err
	}

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

// IsCaRenewable returns true if CA certificate is managed by the MicroOVN
// and therefore valid for automatic renewal
func IsCaRenewable(ctx context.Context, s state.State) (bool, error) {
	var err error
	var CaCertRenewable *database.ConfigItem
	isRenewable := true

	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		CaCertRenewable, err = database.GetConfigItem(ctx, tx, CAAutoRenew)
		if err != nil {
			// For backwards compatibility with older installation that do not have
			// the auto-renew config key in the database, we assume that the CA certificate
			// is generated by the MicroOVN and therefore it's automatically renewable
			switch err.(type) {
			case api.StatusError:
				return nil
			default:
				return err
			}
		}

		if CaCertRenewable.Value == CaIsNotRenewable {
			isRenewable = false
		}
		return nil
	})

	return isRenewable, err
}

// GetCA pulls PEM encoded CA certificate and private key from shared database and returns
// them as parsed objects x509.Certificate and ecdsa.PrivateKey (+ error if any occurred).
func GetCA(ctx context.Context, s state.State) (*x509.Certificate, any, error) {
	var err error
	var CACertRecord *database.ConfigItem
	var CAKeyRecord *database.ConfigItem

	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
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
	caKey, err := parsePrivateKey(keyData.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA private key: %s", err)
	}

	return caCert, caKey, nil
}

// GenerateNewServiceCertificate creates new certificate, signs it with CA certificate stored in the shared database
// and writes resulting certificate and private key to files specified by certPath and keyPath arguments.
// String from serviceName argument will be inserted in certificate's OU and is meant to more easily distinguish
// between multiple certificates with same CN.
func GenerateNewServiceCertificate(ctx context.Context, s state.State, serviceName string, certType CertificateType) error {
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

	caCert, caKey, err := GetCA(ctx, s)
	if err != nil {
		return err
	}

	cert, key, err := issueCertificate(s.Name(), serviceName, certType, caCert, caKey)
	if err != nil {
		return fmt.Errorf("failed to issue certificate for %s: %w", serviceName, err)
	}

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
	case "client":
		certPath, keyPath = paths.PkiClientCertFiles()
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
		err = fmt.Errorf("unknown service '%s'. can't generate certificate paths", service)
	}

	return certPath, keyPath, err
}

// parsePrivateKey attempts to parse raw bytes of the private key in multiple formats:
//   - PKCS8
//   - PKCS1
//   - EC
//
// Since crypto/x509 package doesn't seem to have general "ParsePrivateKey" function, we
// need to manually try parsing the key from any format that we intend to support.
func parsePrivateKey(rawPrivateKey []byte) (any, error) {
	parsedPrivateKey, err := x509.ParsePKCS8PrivateKey(rawPrivateKey)
	if err == nil {
		return parsedPrivateKey, nil
	}
	parsedPrivateKey, err = x509.ParseECPrivateKey(rawPrivateKey)
	if err == nil {
		return parsedPrivateKey, nil
	}
	parsedPrivateKey, err = x509.ParsePKCS1PrivateKey(rawPrivateKey)
	if err == nil {
		return parsedPrivateKey, nil
	}
	return nil, err
}
