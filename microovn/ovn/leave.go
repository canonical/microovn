package ovn

import (
	"context"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/node"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/snap"
)

// Leave function gracefully departs from the OVN cluster before the member is removed from MicroOVN
// cluster. It ensures that:
//   - OVN chassis is stopped and removed from SB database
//   - OVN NB cluster is cleanly departed
//   - OVN SB cluster is cleanly departed
//
// Note (mkalcok): At this point, database table `services` no longer contains entries
// for departing cluster member, so we'll try to exit/leave/stop all possible services
// ignoring any errors from services that are not actually running.
func Leave(ctx context.Context, s state.State, _ bool) error {
	var err error
	chassisName := s.Name()

	// Gracefully exit OVN controller causing chassis to be automatically removed.
	logger.Infof("Stopping OVN Controller and removing Chassis '%s' from OVN SB database.", chassisName)
	_, err = ovnCmd.ControllerCtl(ctx, s, "exit")
	if err != nil {
		logger.Warnf("Failed to gracefully stop OVN Controller: %s", err)
	}

	err = snap.Stop("chassis", true)
	if err != nil {
		logger.Warnf("Failed to stop Chassis service: %s", err)
	}

	err = snap.Stop("switch", true)
	if err != nil {
		logger.Warnf("Failed to stop Switch service: %s", err)
	}

	err = node.StopCentral(ctx, s)
	if err != nil {
		logger.Warnf("Failed to stop Central service: %s", err)
	}

	logger.Info("Cleaning up runtime and data directories.")
	err = cleanupPaths()
	if err != nil {
		logger.Warn(err.Error())
	}

	return nil
}
