package ovn

import (
	"fmt"
	"os"
	"path/filepath"
)

func createPaths() error {
	// Create our various paths.
	paths := []string{
		filepath.Join(os.Getenv("SNAP_COMMON"), "run"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "run", "central"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "run", "chassis"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "run", "switch"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "data"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "data", "central"),
		filepath.Join(os.Getenv("SNAP_COMMON"), "data", "central", "db"),
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
