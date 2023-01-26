package ovn

import (
	"fmt"
	"os"
	"path/filepath"
)

func createPaths() error {
	// Create our various paths.
	paths := []string{
		filepath.Join(os.Getenv("SNAP_DATA"), "run"),
		filepath.Join(os.Getenv("SNAP_DATA"), "run", "chassis"),
		filepath.Join(os.Getenv("SNAP_DATA"), "run", "switch"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "data"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "data", "switch"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "data", "switch", "db"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "data", "switch", "openvswitch"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "logs"),
	}

	for _, path := range paths {
		err := os.MkdirAll(path, 0700)
		if err != nil {
			return fmt.Errorf("Unable to create %q: %w", path, err)
		}
	}

	return nil
}
