package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/ovn/paths"
)

// DefaultDBConnectWait - Default time to wait for connection to ovsdb.
const DefaultDBConnectWait = 30

// OvsdbConnected       - String representing the connected state.
const OvsdbConnected = "connected"

// OvsdbRemoved         - String representing the removed state.
const OvsdbRemoved = "removed"

// OvsdbSpec is a helper structure that encapsulates properties of an OVN/OVS database.
type OvsdbSpec struct {
	SocketURL    string // URL to an open database socket (e.g "unix:/path/db.sock")
	Name         string // Name of the database within the db file (e.g. "OVN_Northbound")
	ShortName    string // Shorthand name for the database (e.g. "nb")
	FriendlyName string // Human friendly name of the database ideal for logging purposes (e.g. "Northbound")
	Schema       string // Path to a schema file for the database
	IsCentral    bool   // Whether the database is used by OVN central services
}

// OvsdbType is an enumeration of valid types of ovsdb databases which this package recognizes
// and can execute commands against.
type OvsdbType int

const (
	// OvsdbTypeNBLocal     - OVSDB Database with schema OVN_Northbound.
	OvsdbTypeNBLocal OvsdbType = iota
	// OvsdbTypeSBLocal     - OVSDB Database with schema OVN_Southbound.
	OvsdbTypeSBLocal
	// OvsdbTypeSwitchLocal - OVSDB Database with schema Open_vSwitch.
	OvsdbTypeSwitchLocal
)

// NewOvsdbSpec is a helper function that takes OvsdbType as an argument and generates
// proper OvsdbSpec for given type.
func NewOvsdbSpec(dbType OvsdbType) (*OvsdbSpec, error) {
	var dbSpec *OvsdbSpec
	var err error

	switch dbType {
	case OvsdbTypeNBLocal:
		dbSpec = &OvsdbSpec{
			SocketURL:    fmt.Sprintf("unix:%s", paths.OvnNBDatabaseSock()),
			Schema:       paths.OvsdbNbSchema(),
			Name:         "OVN_Northbound",
			FriendlyName: "Northbound",
			ShortName:    "nb",
			IsCentral:    true,
		}
	case OvsdbTypeSBLocal:
		dbSpec = &OvsdbSpec{
			SocketURL:    fmt.Sprintf("unix:%s", paths.OvnSBDatabaseSock()),
			Schema:       paths.OvsdbSbSchema(),
			Name:         "OVN_Southbound",
			FriendlyName: "Southbound",
			ShortName:    "sb",
			IsCentral:    true,
		}
	case OvsdbTypeSwitchLocal:
		dbSpec = &OvsdbSpec{
			SocketURL:    fmt.Sprintf("unix:%s", paths.OvsDatabaseSock()),
			Schema:       paths.OvsdbSwitchSchema(),
			Name:         "Open_vSwitch",
			FriendlyName: "OpenvSwitch",
			ShortName:    "switch",
			IsCentral:    false,
		}
	default:
		err = errors.New("unknown ovsdb type")
	}

	return dbSpec, err
}

// WaitForDBState as the name suggests, waits for specified ovsdb database to settle in
// specified state. If database does not reach this state within timeout, this function returns error.
// SocketURL specified in "db" parameter does not need to necessarily exist before this function is executed,
// creation of the database socket (db.SocketURL) will be awaited as well.
func WaitForDBState(ctx context.Context, _ state.State, db *OvsdbSpec, dbState string, timeout int) error {
	_, err := shared.RunCommandContext(
		ctx,
		"ovsdb-client",
		"--timeout",
		strconv.Itoa(timeout),
		"wait",
		db.SocketURL,
		db.Name,
		dbState,
	)
	if err != nil {
		return fmt.Errorf("database in '%s' (%s) failed to reach state '%s': %w", db.Name, db.SocketURL, dbState, err)
	}
	return nil
}

// ovnDBCtl is a helper function to execute "ovn-nbctl" and "ovn-sbctl" commands
// which are re-tried up to 3 times. If command arguments do not specify timeout (-t or
// --timeout), a default of 30s will be added automatically. It takes "dbType"
// parameter that identifies which database server it's going to talk to and
// "args" parameters which is list of arguments that are directly passed to
// ovn-nbctl/ovn-sbctl commands. Before the command is executed, this function
// ensures that underlying database is in connected state. If the database does
// not reach "connected" state before specified timeout, an error is returned.
func ovnDBCtl(ctx context.Context, s state.State, dbType OvsdbType, timeout int, args ...string) (string, error) {
	var baseCmd string

	switch dbType {
	case OvsdbTypeNBLocal:
		baseCmd = "ovn-nbctl"
	case OvsdbTypeSBLocal:
		baseCmd = "ovn-sbctl"
	default:
		return "", errors.New("unknown DB type. OVN commands work only with NB or SB database")
	}

	dbSpec, err := NewOvsdbSpec(dbType)
	if err != nil {
		return "", err
	}

	err = WaitForDBState(ctx, s, dbSpec, OvsdbConnected, timeout)
	if err != nil {
		return "", err
	}

	if !slices.Contains(args, "--timeout") && !slices.Contains(args, "-t") {
		args = append([]string{"--timeout", "30"}, args...)
	}

	// try command 3 times if it is failing
	for attempts := 0; attempts < 3; attempts++ {
		var output string
		output, err = shared.RunCommandContext(ctx, baseCmd, args...)
		if err == nil {
			return output, nil
		}
	}
	return "", err
}

// NBCtl is a convenience function for execution of ovn-nbctl command against
// OVN NB local unix socket. The command is re-tried up to 3 times.
// If command arguments do not specify timeout (-t or
// --timeout), a default of 30s will be added automatically. Parameter "args"
// is a list of arguments that are passed directly to the shell command.
// Before the command is executed, this function ensures that underlying
// database is in connected state. If the database does not reach "connected"
// state before timeout (defined in DefaultDBConnectWait), an error is returned
// and command is not executed.
func NBCtl(ctx context.Context, s state.State, args ...string) (string, error) {
	return ovnDBCtl(ctx, s, OvsdbTypeNBLocal, DefaultDBConnectWait, args...)
}

// NBCtlCluster is a convenience function for execution of ovn-nbctl command
// against OVN NB cluster endpoints. The command is re-tried up to 3 times.
// If command arguments do not specify timeout (-t or
// --timeout), a default of 30s will be added automatically. Parameter "args"
// is a list of arguments that are passed directly to the shell command.
//
// Warning: This function will fail if local MicroOVN node is not bootstrapped.
func NBCtlCluster(ctx context.Context, args ...string) (string, error) {
	if !slices.Contains(args, "--timeout") && !slices.Contains(args, "-t") {
		args = append([]string{"--timeout", "30"}, args...)
	}

	var err error
	// try command 3 times if it is failing
	for attempts := 0; attempts < 3; attempts++ {
		var output string
		output, err = shared.RunCommandContext(ctx, filepath.Join(paths.Wrappers(), "ovn-nbctl"), args...)
		if err == nil {
			return output, nil
		}
	}
	return "", err
}

// SBCtl is a convenience function for execution of ovn-sbctl command against
// OVN NB local unix socket. The command is re-tried up to 3 times.
// If command arguments do not specify timeout (-t or
// --timeout), a default of 30s will be added automatically. Parameter "args" is
// a list of arguments that are passed directly to the shell command. Before the
// command is executed, this function ensures that underlying database is in
// connected state. If the database does not reach "connected" state before
// timeout (defined in DefaultDBConnectWait), an error is returned and command
// is not executed.
func SBCtl(ctx context.Context, s state.State, args ...string) (string, error) {
	return ovnDBCtl(ctx, s, OvsdbTypeSBLocal, DefaultDBConnectWait, args...)
}

// VSCtl is a convenience function for execution of ovs-vsctl command which is
// re-tried up to 3 times. If command arguments do not specify timeout (-t or
// --timeout), a default of 30s will be added automatically. Parameter "args" is
// a list of arguments that are passed directly to the shell command. Before the
// command is executed, this function ensures that underlying database is in
// connected state. If the database does not reach "connected" state before
// timeout (defined in DefaultDBConnectWait), an error is returned and command
// is not executed.
func VSCtl(ctx context.Context, s state.State, args ...string) (string, error) {

	dbSpec, err := NewOvsdbSpec(OvsdbTypeSwitchLocal)
	if err != nil {
		return "", err
	}

	err = WaitForDBState(ctx, s, dbSpec, OvsdbConnected, DefaultDBConnectWait)
	if err != nil {
		return "", err
	}

	if !slices.Contains(args, "--timeout") && !slices.Contains(args, "-t") {
		args = append([]string{"--timeout", "30"}, args...)
	}

	// try command 3 times if it is failing
	for attempts := 0; attempts < 3; attempts++ {
		var output string
		output, err = shared.RunCommandContext(ctx, "ovs-vsctl", args...)
		if err == nil {
			return output, nil
		}
	}
	return "", err
}

// AppCtl is a convenience function that wraps execution of 'ovn-appctl' command. It requires argument
// 'target' which will be substituted to the '-t' argument of 'ovn-appctl'. Rest of the 'args' will be passed
// to the ovn-appctl unchanged.
func AppCtl(ctx context.Context, _ state.State, target string, args ...string) (string, error) {
	arguments := []string{"-t", target}
	arguments = append(arguments, args...)
	return shared.RunCommandContext(
		ctx,
		"ovn-appctl",
		arguments...,
	)
}

// ControllerCtl is a wrapper function that executes 'ovs-appctl' command specifically
// targeted at running OVN Controller process. The '-t' argument of 'ovs-appctl' will be
// configured automatically. Any arguments supplied in 'args' will be passed to the 'ovs-appctl'
// unchanged.
func ControllerCtl(ctx context.Context, _ state.State, args ...string) (string, error) {
	arguments := []string{"-t", "ovn-controller"}
	arguments = append(arguments, args...)

	stdout, _, err := shared.RunCommandSplit(
		ctx,
		append(os.Environ(), fmt.Sprintf("OVS_RUNDIR=%s", paths.OvnRuntimeDir())),
		nil,
		"ovs-appctl",
		arguments...,
	)

	return stdout, err
}

// OvsdbClient is a wrapper function that executes 'ovsdb-client' command. It first ensures that the database
// is connected and returns error if the database is not connected within <connectTimeout> seconds. Then it runs
// "ovsdb-client" command with timeout of <resultTimeout> seconds.
// Argument "args" should contain array of strings with subcommand and other arguments that will be passed directly
// to the "ovsdb-client". Note that it is not necessary to pass "-t" argument, as the timeout is automatically included.
func OvsdbClient(ctx context.Context, s state.State, dbSpec *OvsdbSpec, connectTimeout int, resultTimeout int, args ...string) (string, error) {
	err := WaitForDBState(ctx, s, dbSpec, OvsdbConnected, connectTimeout)
	if err != nil {
		return "", err
	}

	arguments := []string{"-t", strconv.Itoa(resultTimeout)}
	arguments = append(arguments, args...)

	stdout, _, err := shared.RunCommandSplit(
		ctx,
		nil,
		nil,
		"ovsdb-client",
		arguments...,
	)

	return stdout, err
}
