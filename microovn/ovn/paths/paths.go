// Package paths provides path constants and helper functions.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var snapRoot = os.Getenv("SNAP")
var pathRoot = os.Getenv("SNAP_COMMON")
var runtimeDir = filepath.Join(pathRoot, "run")
var dataDir = filepath.Join(pathRoot, "data")

// Root returns $SNAP_COMMON root of MicroOVN
func Root() string {
	return pathRoot
}

// LogsDir returns path to the directory where OVN stores log files
func LogsDir() string {
	return filepath.Join(pathRoot, "logs")
}

// OvnRuntimeDir returns path to the directory where OVN Central creates its runtime files
func OvnRuntimeDir() string {
	return filepath.Join(runtimeDir, "ovn")
}

// CentralDBDir returns path to the directory where OVN Central stores its databases
func CentralDBDir() string {
	return filepath.Join(dataDir, "central", "db")
}

// CentralDBNBPath returns path to the Northbound database file
func CentralDBNBPath() string { return filepath.Join(CentralDBDir(), "ovnnb_db.db") }

// CentralDBSBPath returns path to the Southbound database file
func CentralDBSBPath() string { return filepath.Join(CentralDBDir(), "ovnsb_db.db") }

// CentralDBSBBackupPath returns path to the where the Southbound database file should
// be backed up to
func CentralDBSBBackupPath() string {
	return filepath.Join(CentralDBDir(),
		"ovnsb_db_backup_"+time.Now().Format(time.DateTime)+".db")
}

// CentralDBNBBackupPath returns path to the where the Northbound database file should
// be backed up to
func CentralDBNBBackupPath() string {
	return filepath.Join(CentralDBDir(),
		"ovnnb_db_backup_"+time.Now().Format(time.DateTime)+".db")
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
	return filepath.Join(EnvDir(), "ovn.env")
}

// OvnNBDatabaseSock returns path to the local unix socket used by Northbound OVN database
func OvnNBDatabaseSock() string {
	return filepath.Join(OvnRuntimeDir(), "ovnnb_db.sock")
}

// OvnSBDatabaseSock returns path to the local unix socket used by Southbound OVN database
func OvnSBDatabaseSock() string {
	return filepath.Join(OvnRuntimeDir(), "ovnsb_db.sock")
}

// OvnNBControlSock returns path to the local control socket for Northbound OVN service
func OvnNBControlSock() string {
	return filepath.Join(OvnRuntimeDir(), "ovnnb_db.ctl")
}

// OvnSBControlSock returns path to the local control socket for Southbound OVN service
func OvnSBControlSock() string {
	return filepath.Join(OvnRuntimeDir(), "ovnsb_db.ctl")
}

// OvsDatabaseSock returns path to the local unix socket used by OpenvSwitch database
func OvsDatabaseSock() string {
	return filepath.Join(SwitchRuntimeDir(), "db.sock")
}

// PkiDir returns path to the directory that store OVN certificates
func PkiDir() string {
	return filepath.Join(dataDir, "pki")
}

// EnvDir returns path to the directory that store OVN environment variables.
func EnvDir() string {
	return filepath.Join(dataDir, "env")
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

// PkiClientCertFiles returns paths to certificate and private key used by client
func PkiClientCertFiles() (string, string) {
	return getServiceCertFiles("client")
}

// OvsdbSbSchema returns path to schema file for OVN Southbound database
func OvsdbSbSchema() string { return filepath.Join(snapRoot, "share", "ovn", "ovn-sb.ovsschema") }

// OvsdbNbSchema returns path to schema file for OVN Northbound database
func OvsdbNbSchema() string { return filepath.Join(snapRoot, "share", "ovn", "ovn-nb.ovsschema") }

// OvsdbSwitchSchema returns path to schema file for OpenvSwitch
func OvsdbSwitchSchema() string {
	return filepath.Join(snapRoot, "share", "openvswitch", "vswitch.ovsschema")
}

// Wrappers returns path to a directory with snap's command wrappers
func Wrappers() string { return filepath.Join(snapRoot, "commands") }

// FrrConfigDir returns path to a directory that FRR uses to store configuration
func FrrConfigDir() string {
	return filepath.Join(dataDir, "frr", "etc")
}

// FrrDefaultConfig returns path to FRR's default config file
func FrrDefaultConfig() string {
	return filepath.Join(snapRoot, "etc", "frr", "frr.conf")
}

// FrrStartupConfig returns path to current FRR's startup config
func FrrStartupConfig() string {
	return filepath.Join(FrrConfigDir(), "frr.conf")
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
		OvnRuntimeDir(),
		CentralDBDir(),
		SwitchDBDir(),
		SwitchRuntimeDir(),
		SwitchDataDir(),
		LogsDir(),
		PkiDir(),
		EnvDir(),
		FrrConfigDir(),
	}
}

// BackupDirs returns list of locations that should be backed up before MicroOVN
// data removal.
func BackupDirs() []string {
	return []string{
		dataDir,
		LogsDir(),
		FrrConfigDir(),
	}
}

// GetPath returns the corresponding path for a given name. If the name is not
// recognized, it returns nil.
func GetPath(name string) interface{} {
	switch name {
	case "Root":
		return Root()
	case "LogsDir":
		return LogsDir()
	case "OvnRuntimeDir":
		return OvnRuntimeDir()
	case "CentralDBDir":
		return CentralDBDir()
	case "CentralDBNBPath":
		return CentralDBNBPath()
	case "CentralDBSBPath":
		return CentralDBSBPath()
	case "CentralDBSBBackupPath":
		return CentralDBSBBackupPath()
	case "CentralDBNBBackupPath":
		return CentralDBNBBackupPath()
	case "SwitchDBDir":
		return SwitchDBDir()
	case "SwitchRuntimeDir":
		return SwitchRuntimeDir()
	case "SwitchDataDir":
		return SwitchDataDir()
	case "OvnEnvFile":
		return OvnEnvFile()
	case "OvnNBDatabaseSock":
		return OvnNBDatabaseSock()
	case "OvnSBDatabaseSock":
		return OvnSBDatabaseSock()
	case "OvnNBControlSock":
		return OvnNBControlSock()
	case "OvnSBControlSock":
		return OvnSBControlSock()
	case "OvsDatabaseSock":
		return OvsDatabaseSock()
	case "PkiDir":
		return PkiDir()
	case "EnvDir":
		return EnvDir()
	case "PkiCaCertFile":
		return PkiCaCertFile()
	case "OvsdbSbSchema":
		return OvsdbSbSchema()
	case "OvsdbNbSchema":
		return OvsdbNbSchema()
	case "OvsdbSwitchSchema":
		return OvsdbSwitchSchema()
	default:
		return nil
	}
}
