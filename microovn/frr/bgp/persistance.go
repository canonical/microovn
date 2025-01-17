package bgp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/canonical/microcluster/v2/state"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/zitadel/logging"
)

// EnsureInterfacesInVrf looks up all the OVS interfaces (ports) that are tagged
// by MicroOVN for purpose of BGP redirection, and ensures that these interfaces are in the correct
// VRF, have IPv4 address set and are UP.
func EnsureInterfacesInVrf(ctx context.Context, s state.State) error {
	var allErrors error
	_, err := ovnCmd.NBCtlCluster(ctx, "--wait", "hv", "sync")
	if err != nil {
		logging.Warnf("failed to wait for chassis sync: %v", err)
	}
	bgpRedirectPorts, err := ovnCmd.VSCtl(ctx, s, "--bare", "--columns", "name", "find", "port",
		fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
	)
	if err != nil {
		return fmt.Errorf("failed to lookup BGP redirect ports managed by MicroOVN: %v", err)
	}

	for _, port := range parseOvnFind(bgpRedirectPorts) {
		portVrf, err := ovnCmd.VSCtl(ctx, s, "get", "port", port, fmt.Sprintf("external-ids:%s", BgpVrfTable))
		if err != nil {
			allErrors = errors.Join(allErrors, fmt.Errorf("failed to lookup VRF table of port '%s': %v", port, err))
			continue
		}
		portVrf = strings.TrimSpace(portVrf)

		portIP := ""
		// portIP, err = ovnCmd.VSCtl(ctx, s, "get", "port", port, fmt.Sprintf("external-ids:%s", BgpIfaceIP))
		// if err != nil {
		// 	allErrors = errors.Join(allErrors, fmt.Errorf("failed to lookup IPv4 address of port '%s': %v", port, err))
		// }
		// portIP = strings.Trim(strings.TrimSpace(portIP), "\"")

		err = moveInterfaceToVrf(ctx, port, portIP, portVrf)
		if err != nil {
			allErrors = errors.Join(allErrors, fmt.Errorf("failed to move interface '%s' to VRF '%s': %v", port, portVrf, err))
		}

	}
	return allErrors
}
