package ovn

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/database"
)

var ovnEnvTpl = template.Must(template.New("ovnEnvTpl").Parse(`# # Generated by MicroOVN, DO NOT EDIT.
OVN_INITIAL_NB="{{ .nbInitial }}"
OVN_INITIAL_SB="{{ .sbInitial }}"
OVN_NB_CONNECT="{{ .nbConnect }}"
OVN_SB_CONNECT="{{ .sbConnect }}"
OVN_LOCAL_IP="{{ .localAddr }}"
`))

func connectString(s *state.State, port int) (string, error) {
	var err error
	var servers []database.Service

	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		serviceName := "central"
		servers, err = database.GetServices(ctx, tx, database.ServiceFilter{Service: &serviceName})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	addresses := make([]string, len(servers))
	remotes := s.Remotes().RemotesByName()
	for i, server := range servers {
		remote, ok := remotes[server.Member]
		if !ok {
			continue
		}

		addresses[i] = fmt.Sprintf("tcp:%s:%d", remote.Address.Addr().String(), port)
	}

	return strings.Join(addresses, ","), nil
}

func generateEnvironment(s *state.State) error {
	// Get the servers.
	nbConnect, err := connectString(s, 6641)
	if err != nil {
		return err
	}

	sbConnect, err := connectString(s, 6642)
	if err != nil {
		return err
	}

	// Get the initial (first server).
	var nbInitial string
	var sbInitial string
	err = s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		serviceName := "central"
		servers, err := database.GetServices(ctx, tx, database.ServiceFilter{Service: &serviceName})
		if err != nil {
			return err
		}

		server := servers[0]

		remotes := s.Remotes().RemotesByName()
		remote, ok := remotes[server.Member]
		if !ok {
			return fmt.Errorf("Remote couldn't be found for %q", server.Member)
		}

		nbInitial = remote.Address.Addr().String()
		sbInitial = remote.Address.Addr().String()

		return nil
	})
	if err != nil {
		return err
	}

	// Generate ceph.conf.
	fd, err := os.OpenFile(filepath.Join(os.Getenv("SNAP_COMMON"), "data", "ovn.env"), os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("Couldn't open ovn.env: %w", err)
	}
	defer fd.Close()

	err = ovnEnvTpl.Execute(fd, map[string]any{
		"localAddr": s.Address().Hostname(),
		"nbInitial": nbInitial,
		"sbInitial": sbInitial,
		"nbConnect": nbConnect,
		"sbConnect": sbConnect,
	})
	if err != nil {
		return fmt.Errorf("Couldn't render ovn.env: %w", err)
	}

	return nil
}
