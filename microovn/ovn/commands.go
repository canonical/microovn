package ovn

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/ovn/paths"
)

const defaultDBConnectWait = 30 //Default time to wait for connection to ovsdb
const OvsdbConnected = "connected"
const OvsdbRemoved = "removed"

// ovsdbSpec is a helper structure for precise identification of ovsdb databases. A lot of
// ovn/ovs commands take path to either database file, database socket or process control socket
// along with the database name. This structure can be used for such cases.
type ovsdbSpec struct {
	Target string
	Name   string
}

// OvsdbType is an enumeration of valid types of ovsdb databases which this package recognizes
// and can execute commands against.
type OvsdbType int

const (
	OvsdbTypeNBLocal OvsdbType = iota
	OvsdbTypeSBLocal
	OvsdbTypeSwitchLocal
)

// newOvsdbSpec is a helper function that takes OvsdbType as an argument and generates
// proper ovsdbSpec for given type.
func newOvsdbSpec(dbType OvsdbType) (*ovsdbSpec, error) {
	var dbSpec *ovsdbSpec
	var err error

	if dbType == OvsdbTypeNBLocal {
		dbSpec = &ovsdbSpec{
			Target: paths.OvnNBDatabaseSock(),
			Name:   "OVN_Northbound",
		}
	} else if dbType == OvsdbTypeSBLocal {
		dbSpec = &ovsdbSpec{
			Target: paths.OvnSBDatabaseSock(),
			Name:   "OVN_Southbound",
		}
	} else if dbType == OvsdbTypeSwitchLocal {
		dbSpec = &ovsdbSpec{
			Target: paths.OvsDatabaseSock(),
			Name:   "Open_vSwitch",
		}
	} else {
		err = errors.New("unknown ovsdb type")
	}

	return dbSpec, err
}

// waitForDBState as the name suggests, waits for specified ovsdb database to settle in
// specified state. If database does not reach this state within timeout, this function returns error.
// Target specified in "db" parameter does not need to necessarily exist before this function is executed,
// creation of the database socket (db.Target) will be awaited as well.
func waitForDBState(s *state.State, db *ovsdbSpec, dbState string, timeout int) error {
	_, err := shared.RunCommandContext(
		s.Context,
		"ovsdb-client",
		"--timeout",
		strconv.Itoa(timeout),
		"wait",
		fmt.Sprintf("unix:%s", db.Target),
		db.Name,
		dbState,
	)
	if err != nil {
		return fmt.Errorf("database in '%s' (%s) failed to reach state '%s': %w", db.Name, db.Target, dbState, err)
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

	if dbType == OvsdbTypeNBLocal {
		baseCmd = "ovn-nbctl"
	} else if dbType == OvsdbTypeSBLocal {
		baseCmd = "ovn-sbctl"
	} else {
		return "", errors.New("unknown DB type. OVN commands work only with NB or SB database")
	}

	dbSpec, err := newOvsdbSpec(dbType)
	if err != nil {
		return "", err
	}

	err = waitForDBState(s, dbSpec, OvsdbConnected, timeout)
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
	return ovnDBCtl(s, OvsdbTypeNBLocal, defaultDBConnectWait, args...)
}

// SBCtl is a convenience function for execution of ovn-sbctl command. Parameter "args" is list of arguments
// that are passed directly to the shell command. Before the command is executed, this
// function ensures that underlying database is in connected state. If the database does not reach "connected"
// state before timeout (defined in defaultDBConnectWait), an error is returned and command is not executed.
func SBCtl(s *state.State, args ...string) (string, error) {
	return ovnDBCtl(s, OvsdbTypeSBLocal, defaultDBConnectWait, args...)
}

// VSCtl is a convenience function for execution of ovs-vsctl command. Parameter "args" is list of arguments
// that are passed directly to the shell command. Before the command is executed, this
// function ensures that underlying database is in connected state. If the database does not reach "connected"
// state before timeout (defined in defaultDBConnectWait), an error is returned and command is not executed.
func VSCtl(s *state.State, args ...string) (string, error) {

	dbSpec, err := newOvsdbSpec(OvsdbTypeSwitchLocal)
	if err != nil {
		return "", err
	}

	err = waitForDBState(s, dbSpec, OvsdbConnected, defaultDBConnectWait)
	if err != nil {
		return "", err
	}

	return shared.RunCommandContext(s.Context, "ovs-vsctl", args...)
}

// AppCtl is a convenience function that wraps execution of 'ovn-appctl' command. It requires argument
// 'target' which will be substituted to the '-t' argument of 'ovn-appctl'. Rest of the 'args' will be passed
// to the ovn-appctl unchanged.
func AppCtl(s *state.State, target string, args ...string) (string, error) {
	arguments := []string{"-t", target}
	arguments = append(arguments, args...)
	return shared.RunCommandContext(
		s.Context,
		"ovn-appctl",
		arguments...,
	)
}

// ControllerCtl is a wrapper function that executes 'ovs-appctl' command specifically
// targeted at running OVN Controller process. The '-t' argument of 'ovs-appctl' will be
// configured automatically. Any arguments supplied in 'args' will be passed to the 'ovs-appctl'
// unchanged.
func ControllerCtl(s *state.State, args ...string) (string, error) {
	arguments := []string{"-t", "ovn-controller"}
	arguments = append(arguments, args...)

	stdout, _, err := shared.RunCommandSplit(
		s.Context,
		append(os.Environ(), fmt.Sprintf("OVS_RUNDIR=%s", paths.OvnRuntimeDir())),
		nil,
		"ovs-appctl",
		arguments...,
	)

	return stdout, err
}

// GetOvsdbLocalPath returns path to the database file or local unix socket based on the supplied "dbType"
func GetOvsdbLocalPath(dbType OvsdbType) (string, error) {
	spec, err := newOvsdbSpec(dbType)
	return spec.Target, err
}
