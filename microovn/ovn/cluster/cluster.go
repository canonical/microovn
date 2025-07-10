// Package cluster builds on top of microovn.ovn.cmd package and provides
// functions for commonly used actions performed on the OVN cluster and related
// services. It removes the necessity to manually build up ovn CLI commands.
package cluster

import (
	"context"
	"fmt"

	"github.com/canonical/microcluster/v2/state"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/environment"
)

// UpdateOvnListenConfig configures the OVN NB and SB databases to listen on the appropriate ports.
func UpdateOvnListenConfig(ctx context.Context, s state.State) error {
	nbDB, err := ovnCmd.NewOvsdbSpec(ovnCmd.OvsdbTypeNBLocal)
	if err != nil {
		return fmt.Errorf("failed to get path to OVN NB database socket: %w", err)
	}
	sbDB, err := ovnCmd.NewOvsdbSpec(ovnCmd.OvsdbTypeSBLocal)
	if err != nil {
		return fmt.Errorf("failed to get path to OVN SB database socket: %w", err)
	}

	protocol := environment.NetworkProtocol(ctx, s)
	_, err = ovnCmd.NBCtl(
		ctx,
		s,
		"--no-leader-only",
		fmt.Sprintf("--db=%s", nbDB.SocketURL),
		"set-connection",
		fmt.Sprintf("p%s:6641:[::]", protocol),
	)
	if err != nil {
		return fmt.Errorf("error setting ovn NB connection string: %s", err)
	}

	_, err = ovnCmd.SBCtl(
		ctx,
		s,
		"--no-leader-only",
		fmt.Sprintf("--db=%s", sbDB.SocketURL),
		"set-connection",
		fmt.Sprintf("p%s:6642:[::]", protocol),
	)
	if err != nil {
		return fmt.Errorf("error setting ovn SB connection string: %s", err)
	}

	return nil
}
