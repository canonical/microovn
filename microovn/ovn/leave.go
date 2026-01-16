package ovn

import (
	"context"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v3/state"

	"github.com/canonical/microovn/microovn/node"
	"github.com/canonical/microovn/microovn/ovn/environment"
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
	// Attempt to disable each service
	err := node.DisableAllServices(ctx, s)
	if err != nil {
		return err
	}

	logger.Info("Cleaning up runtime and data directories.")
	err = environment.CleanupPaths()
	if err != nil {
		logger.Warn(err.Error())
	}

	return nil
}
