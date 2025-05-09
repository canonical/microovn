package ovn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/cluster"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/ovn/certificates"
	"github.com/canonical/microovn/microovn/ovn/paths"
)

var ovnEnvTpl = template.Must(template.New("ovnEnvTpl").Parse(`# # Generated by MicroOVN, DO NOT EDIT.
OVN_INITIAL_NB="{{ .nbInitial }}"
OVN_INITIAL_SB="{{ .sbInitial }}"
OVN_NB_CONNECT="{{ .nbConnect }}"
OVN_SB_CONNECT="{{ .sbConnect }}"
OVN_LOCAL_IP="{{ .localAddr }}"
`))

// networkProtocol returns appropriate network protocol that should be used
// by OVN services.
func networkProtocol(ctx context.Context, s state.State) string {
	_, _, err := certificates.GetCA(ctx, s)
	if err != nil {
		return "tcp"
	}
	return "ssl"
}

// Builds environment variable strings for OVN.
func environmentString(ctx context.Context, s state.State, port int) (string, string, error) {
	var err error
	var servers []database.Service
	var clusterMap map[string]cluster.CoreClusterMember
	err = s.Database().Transaction(ctx, func(ctx context.Context, tx *sql.Tx) error {
		serviceName := "central"
		servers, err = database.GetServices(ctx, tx, database.ServiceFilter{Service: &serviceName})
		if err != nil {
			return err
		}

		clusterMembers, err := cluster.GetCoreClusterMembers(ctx, tx)
		if err != nil {
			return err
		}

		clusterMap = make(map[string]cluster.CoreClusterMember, len(clusterMembers))
		for _, clusterMember := range clusterMembers {
			clusterMap[clusterMember.Name] = clusterMember
		}

		return nil
	})
	if err != nil {
		return "", "", err
	}

	addresses := make([]string, 0, len(servers))
	var initialString string
	protocol := networkProtocol(ctx, s)
	for i, server := range servers {
		member := clusterMap[server.Member]
		memberAddr, err := netip.ParseAddrPort(member.Address)
		if err != nil {
			return "", "", err
		}

		if i == 0 {
			initialString = memberAddr.Addr().String()
			if memberAddr.Addr().Is6() {
				initialString = "[" + initialString + "]"
			}
		}

		addresses = append(
			addresses,
			fmt.Sprintf("%s:%s",
				protocol,
				net.JoinHostPort(memberAddr.Addr().String(), strconv.Itoa(port)),
			),
		)
	}

	return strings.Join(addresses, ","), initialString, nil
}

func generateEnvironment(ctx context.Context, s state.State) error {
	// Get the servers.
	nbConnect, nbInitial, err := environmentString(ctx, s, 6641)
	if err != nil {
		return err
	}

	sbConnect, sbInitial, err := environmentString(ctx, s, 6642)
	if err != nil {
		return err
	}

	// Generate ovn.env.
	fd, err := os.OpenFile(paths.OvnEnvFile(), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("couldn't open ovn.env: %w", err)
	}
	defer fd.Close()

	localAddr := s.Address().Hostname()
	if ip, err := netip.ParseAddr(localAddr); err == nil && ip.Is6() {
		localAddr = "[" + localAddr + "]"
	}

	//set northbound to be at the local address if there is no central db found
	if nbInitial == "" {
		nbInitial = localAddr
	}
	if nbConnect == "" {
		nbConnect = "ssl:" + localAddr + ":6641"
	}

	//set southbound to be at the local address if there is no central db found
	if sbInitial == "" {
		sbInitial = localAddr
	}
	if sbConnect == "" {
		sbConnect = "ssl:" + localAddr + ":6642"
	}

	err = ovnEnvTpl.Execute(fd, map[string]any{
		"localAddr": localAddr,
		"nbInitial": nbInitial,
		"sbInitial": sbInitial,
		"nbConnect": nbConnect,
		"sbConnect": sbConnect,
	})
	if err != nil {
		return fmt.Errorf("couldn't render ovn.env: %w", err)
	}

	return nil
}

func createPaths() error {
	// Create our various paths.
	for _, path := range paths.RequiredDirs() {
		err := os.MkdirAll(path, 0700)
		if err != nil {
			return fmt.Errorf("unable to create %q: %w", path, err)
		}
	}

	return nil
}

// cleanupPaths backs up directories defined by paths.BackupDirs and then removes directories
// created by createPaths function. This effectively removes any data created during MicroOVN runtime.
func cleanupPaths() error {
	var errs []error

	// Create timestamped backup dir
	backupDir := fmt.Sprintf("backup_%d", time.Now().Unix())
	backupPath := filepath.Join(paths.Root(), backupDir)
	err := os.Mkdir(backupPath, 0750)
	if err != nil {
		errs = append(
			errs,
			fmt.Errorf(
				"failed to create backup directory '%s', refusing to continue with data removal: %s",
				backupPath,
				err,
			),
		)
		return errors.Join(errs...)
	}

	// Backup selected directories
	for _, dir := range paths.BackupDirs() {
		_, fileName := filepath.Split(dir)
		destination := filepath.Join(backupPath, fileName)
		err = os.Rename(dir, destination)
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Return if any backups failed
	if len(errs) > 0 {
		errs = append(
			errs,
			fmt.Errorf("failures occured during backup, refusing to continue with data removal"),
		)
		return errors.Join(errs...)
	}
	logger.Infof("MicroOVN data backed up to %s", backupPath)

	// Remove rest of the directories
	for _, dir := range paths.RequiredDirs() {
		err = os.RemoveAll(dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to remove directory '%s': %w", dir, err))
		}
	}

	return errors.Join(errs...)
}
