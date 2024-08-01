package ovn

import (
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/state"

	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/paths"
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
func Leave(s *state.State, force bool) error {
	var err error
	chassisName := s.Name()

	// Gracefully exit OVN controller causing chassis to be automatically removed.
	logger.Infof("Stopping OVN Controller and removing Chassis '%s' from OVN SB database.", chassisName)
	_, err = ovnCmd.ControllerCtl(s, "exit")
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

	// Leave SB and NB clusters
	logger.Info("Leaving OVN Northbound cluster")
	_, err = ovnCmd.AppCtl(s, paths.OvnNBControlSock(), "cluster/leave", "OVN_Northbound")
	if err != nil {
		logger.Warnf("Failed to leave OVN Northbound cluster: %s", err)
	}

	logger.Info("Leaving OVN Southbound cluster")
	_, err = ovnCmd.AppCtl(s, paths.OvnSBControlSock(), "cluster/leave", "OVN_Southbound")
	if err != nil {
		logger.Warnf("Failed to leave OVN Southbound cluster: %s", err)
	}

	// Wait for NB and SB cluster members to complete departure process
	nbDatabase, err := ovnCmd.NewOvsdbSpec(ovnCmd.OvsdbTypeNBLocal)
	if err == nil {
		err = ovnCmd.WaitForDBState(s, nbDatabase, ovnCmd.OvsdbRemoved, ovnCmd.DefaultDBConnectWait)
		if err != nil {
			logger.Warnf("Failed to wait for NB cluster departure: %s", err)
		}
	} else {
		logger.Warnf("Failed to get NB database specification: %s", err)
	}

	sbDatabase, err := ovnCmd.NewOvsdbSpec(ovnCmd.OvsdbTypeSBLocal)
	if err == nil {
		err = ovnCmd.WaitForDBState(s, sbDatabase, ovnCmd.OvsdbRemoved, ovnCmd.DefaultDBConnectWait)
		if err != nil {
			logger.Warnf("Failed to wait for SB cluster departure: %s", err)
		}
	} else {
		logger.Warnf("Failed to get SB database specification: %s", err)
	}

	err = snap.Stop("ovn-northd", true)
	if err != nil {
		logger.Warnf("Failed to stop OVN northd service: %s", err)
	}

	err = snap.Stop("ovn-ovsdb-server-nb", true)
	if err != nil {
		logger.Warnf("Failed to stop OVN NB service: %s", err)
	}

	err = snap.Stop("ovn-ovsdb-server-sb", true)
	if err != nil {
		logger.Warnf("Failed to stop OVN SB service: %s", err)
	}

	logger.Info("Cleaning up runtime and data directories.")
	err = cleanupPaths()
	if err != nil {
		logger.Warn(err.Error())
	}

	return nil
}
