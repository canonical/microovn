package paths

import (
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
	}
}
