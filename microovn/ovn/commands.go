package ovn

import (
	"errors"
	"fmt"
	"github.com/canonical/microcluster/state"
	"github.com/lxc/lxd/shared"
	"os"
	"path/filepath"
	"strconv"
)

const defaultDBConnectWait = 30 //Default time to wait for connection to ovsdb

// ovsdbSpec is a helper structure for precise identification of ovsdb databases. A lot of
// ovn/ovs commands take path to either database file, database socket or process control socket
// along with the database name. This structure can be used for such cases.
type ovsdbSpec struct {
	Path string
	Name string
}

// OvsdbType is an enumeration of valid types of ovsdb databases which this package recognizes
// and can execute commands against.
type OvsdbType int

const (
	OvsdbTypeNB OvsdbType = iota
	OvsdbTypeSB
	OvsdbTypeSwitch
)

// newOvsdbSpec is a helper function that takes OvsdbType as an argument and generates
// proper ovsdbSpec for given type.
func newOvsdbSpec(dbType OvsdbType) (*ovsdbSpec, error) {
	var dbSpec *ovsdbSpec
	var err error

	if dbType == OvsdbTypeNB {
		dbSpec = &ovsdbSpec{
			Path: filepath.Join(os.Getenv("SNAP_COMMON"), "run", "central", "ovnnb_db.sock"),
			Name: "OVN_Northbound",
		}
	} else if dbType == OvsdbTypeSB {
		dbSpec = &ovsdbSpec{
			Path: filepath.Join(os.Getenv("SNAP_COMMON"), "run", "central", "ovnsb_db.sock"),
			Name: "OVN_Southbound",
		}
	} else if dbType == OvsdbTypeSwitch {
		dbSpec = &ovsdbSpec{
			Path: filepath.Join(os.Getenv("SNAP_COMMON"), "run", "switch", "db.sock"),
			Name: "Open_vSwitch",
		}
	} else {
		err = errors.New("unknown ovsdb type")
	}

	return dbSpec, err
}

// waitForDBConnected as the name suggests, waits for specified ovsdb database to settle in
// "connected" state. If database does not reach this state within timeout, this function returns error.
// Path specified in "db" parameter does not need to necessarily exist before this function is executed,
// creation of the database socket (db.Path) will be awaited as well.
func waitForDBConnected(s *state.State, db *ovsdbSpec, timeout int) error {
	_, err := shared.RunCommandContext(
		s.Context,
		"ovsdb-client",
		"--timeout",
		strconv.Itoa(timeout),
		"wait",
		fmt.Sprintf("unix:%s", db.Path),
		db.Name,
		"connected",
	)
	if err != nil {
		return fmt.Errorf("failed to connect to %s database in '%s': %w", db.Name, db.Path, err)
	}
	return nil
}

// ovnDBCtl is a helper function to execute "ovn-nbctl" and "ovn-sbctl" commands. It takes "dbType" parameter
// that identifies which database server it's going to talk to and "args" parameters which is list of
// arguments that are directly passed to ovn-nbctl/ovn-sbctl commands. Before the command is executed, this
// function ensures that underlying database is in connected state. If the database does not reach "connected"
// state before specified timeout, an error is returned.
func ovnDBCtl(s *state.State, dbType OvsdbType, timeout int, args ...string) (string, error) {
	var baseCmd string

	if dbType == OvsdbTypeNB {
		baseCmd = "ovn-nbctl"
	} else if dbType == OvsdbTypeSB {
		baseCmd = "ovn-sbctl"
	} else {
		return "", errors.New("unknown DB type. OVN commands work only with NB or SB database")
	}

	dbSpec, err := newOvsdbSpec(dbType)
	if err != nil {
		return "", err
	}

	err = waitForDBConnected(s, dbSpec, timeout)
	if err != nil {
		return "", err
	}

	return shared.RunCommandContext(s.Context, baseCmd, args...)
}

// NBCtl is a convenience function for execution of ovn-nbctl command. Parameter "args" is list of arguments
// that are passed directly to the shell command. Before the command is executed, this
// function ensures that underlying database is in connected state. If the database does not reach "connected"
// state before timeout (defined in defaultDBConnectWait), an error is returned and command is not executed.
func NBCtl(s *state.State, args ...string) (string, error) {
	return ovnDBCtl(s, OvsdbTypeNB, defaultDBConnectWait, args...)
}

// SBCtl is a convenience function for execution of ovn-sbctl command. Parameter "args" is list of arguments
// that are passed directly to the shell command. Before the command is executed, this
// function ensures that underlying database is in connected state. If the database does not reach "connected"
// state before timeout (defined in defaultDBConnectWait), an error is returned and command is not executed.
func SBCtl(s *state.State, args ...string) (string, error) {
	return ovnDBCtl(s, OvsdbTypeSB, defaultDBConnectWait, args...)
}

// VSCtl is a convenience function for execution of ovs-vsctl command. Parameter "args" is list of arguments
// that are passed directly to the shell command. Before the command is executed, this
// function ensures that underlying database is in connected state. If the database does not reach "connected"
// state before timeout (defined in defaultDBConnectWait), an error is returned and command is not executed.
func VSCtl(s *state.State, args ...string) (string, error) {

	dbSpec, err := newOvsdbSpec(OvsdbTypeSwitch)
	if err != nil {
		return "", err
	}

	err = waitForDBConnected(s, dbSpec, defaultDBConnectWait)
	if err != nil {
		return "", err
	}

	return shared.RunCommandContext(s.Context, "ovs-vsctl", args...)
}

// GetOvsdbLocalPath returns path to the database file or local unix socket based on the supplied "dbType"
func GetOvsdbLocalPath(dbType OvsdbType) (string, error) {
	spec, err := newOvsdbSpec(dbType)
	return spec.Path, err
}
