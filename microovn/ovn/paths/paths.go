package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

var pathRoot = os.Getenv("SNAP_COMMON")
var runtimeDir = filepath.Join(pathRoot, "run")
var dataDir = filepath.Join(pathRoot, "data")

// LogsDir returns path to the directory where OVN stores log files
func LogsDir() string {
	return filepath.Join(pathRoot, "logs")
}

// CentralRuntimeDir returns path to the directory where OVN Central creates its runtime files
func CentralRuntimeDir() string {
	return filepath.Join(runtimeDir, "central")
}

// CentralDBDir returns path to the directory where OVN Central stores its databases
func CentralDBDir() string {
	return filepath.Join(dataDir, "central", "db")
}

// ChassisRuntimeDir returns path to the directory where OVN Controller stores its runtime files
func ChassisRuntimeDir() string {
	return filepath.Join(runtimeDir, "chassis")
}

// SwitchDBDir returns path to the directory where OpenvSwitch stores its database
func SwitchDBDir() string {
	return filepath.Join(dataDir, "switch", "db")
}

// SwitchRuntimeDir returns path to the directory where OpenvSwitch stores its runtime files
func SwitchRuntimeDir() string {
	return filepath.Join(runtimeDir, "switch")
}

// SwitchDataDir returns path to the directory where OpenvSwitch stores general data files
func SwitchDataDir() string {
	return filepath.Join(dataDir, "switch", "openvswitch")
}

// OvnEnvFile returns path to the file used to configure env variables for OVN commands
func OvnEnvFile() string {
	return filepath.Join(dataDir, "ovn.env")
}

// OvnNBDatabaseSock returns path to the local unix socket used by Northbound OVN database
func OvnNBDatabaseSock() string {
	return filepath.Join(CentralRuntimeDir(), "ovnnb_db.sock")
}

// OvnSBDatabaseSock returns path to the local unix socket used by Southbound OVN database
func OvnSBDatabaseSock() string {
	return filepath.Join(CentralRuntimeDir(), "ovnsb_db.sock")
}

// OvsDatabaseSock returns path to the local unix socket used by OpenvSwitch database
func OvsDatabaseSock() string {
	return filepath.Join(SwitchRuntimeDir(), "db.sock")
}

// PkiDir returns path to the directory that store OVN certificates
func PkiDir() string {
	return filepath.Join(dataDir, "pki")
}

// PkiCaCertFile returns path to CA certificate file
func PkiCaCertFile() string {
	return filepath.Join(PkiDir(), "cacert.pem")
}

// PkiOvnNbCertFiles returns paths to certificate and private key used by OVN Northbound service
func PkiOvnNbCertFiles() (string, string) {
	return getServiceCertFiles("ovnnb")
}

// PkiOvnSbCertFiles returns paths to certificate and private key used by OVN Southbound service
func PkiOvnSbCertFiles() (string, string) {
	return getServiceCertFiles("ovnsb")
}

// PkiOvnNorthdCertFiles returns paths to certificate and private key used by OVN Northd service
func PkiOvnNorthdCertFiles() (string, string) {
	return getServiceCertFiles("ovn-northd")
}

// PkiOvnControllerCertFiles returns paths to certificate and private key used by OVN Controller
func PkiOvnControllerCertFiles() (string, string) {
	return getServiceCertFiles("ovn-controller")
}

func PkiClientCertFiles() (string, string) {
	return getServiceCertFiles("client")
}

// getServiceCertFiles returns path to certificate and key of give service in format
// "<base_dir>/<service_name>-{cert,privkey}.pem"
func getServiceCertFiles(service string) (string, string) {
	cert := filepath.Join(PkiDir(), fmt.Sprintf("%s-cert.pem", service))
	key := filepath.Join(PkiDir(), fmt.Sprintf("%s-privkey.pem", service))
	return cert, key

}

// RequiredDirs returns list of all directories that need to be created for MicroOVN to
// function properly
func RequiredDirs() []string {
	return []string{
		CentralRuntimeDir(),
		CentralDBDir(),
		ChassisRuntimeDir(),
		SwitchDBDir(),
		SwitchRuntimeDir(),
		SwitchDataDir(),
		LogsDir(),
		PkiDir(),
	}
}
