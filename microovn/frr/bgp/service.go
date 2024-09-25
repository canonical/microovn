package bgp

import (
	"context"
	"errors"
	"fmt"

	"github.com/canonical/microcluster/v2/state"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/snap"
	"github.com/zitadel/logging"
)

// FrrBgpService - Name of the FRR BGP service managed by MicroOVN
const FrrBgpService = "frr-bgp"

// FrrZebraService - Name of the FRR Zebra service managed by MicroOVN
const FrrZebraService = "frr-zebra"

// EnableService starts BGP service managed by MicroOVN. If external connections are specified in the
// "extraConfig" parameter, it also sets up additional OVS ports (one for each external connection) and
// redirects BGP+BFD traffic from the external networks to them.
func EnableService(ctx context.Context, s state.State, extraConfig *types.ExtraBgpConfig) error {
	if extraConfig != nil {
		err := extraConfig.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate BGP config. Services won't be started: %v", err)
		}
	}

	err := snap.Start(FrrZebraService, true)
	if err != nil {
		logging.Errorf("Failed to start %s service: %s", FrrZebraService, err)
		return errors.New("failed to start zebra service")
	}

	err = snap.Start(FrrBgpService, true)
	if err != nil {
		logging.Errorf("Failed to start %s service: %s", FrrBgpService, err)
		return errors.New("failed to start BGP service")
	}

	if extraConfig == nil {
		return nil
	}

	extConnections, err := extraConfig.ParseExternalConnection()
	if err != nil {
		logging.Errorf("Failed to parse external connections: %v", err)
	}

	err = createExternalBridges(ctx, s, extConnections)
	if err != nil {
		return err
	}

	err = createExternalNetworks(ctx, s, extConnections)
	if err != nil {
		return err
	}

	err = createVrf(ctx, s, extConnections, extraConfig.Vrf)
	if err != nil {
		return err
	}

	err = redirectBgp(ctx, s, extConnections, extraConfig.Vrf)
	if err != nil {
		return err
	}

	if extraConfig.Asn != "" {
		err = startBgpUnnumbered(ctx, extConnections, extraConfig.Vrf, extraConfig.Asn)
	}

	return err
}

// DisableService stops and disables BGP services managed by MicroOVN.
func DisableService(ctx context.Context, s state.State) error {
	var allErrors error

	err := snap.Stop(FrrZebraService, true)
	if err != nil {
		logging.Warnf("Failed to stop %s service: %s", FrrZebraService, err)
		allErrors = errors.Join(allErrors, errors.New("failed to stop zebra service"))
	}

	err = snap.Stop(FrrBgpService, true)
	if err != nil {
		logging.Warnf("Failed to stop %s service: %s", FrrBgpService, err)
		allErrors = errors.Join(allErrors, errors.New("failed to stop BGP service"))
	}

	err = teardownAll(ctx, s)
	if err != nil {
		allErrors = errors.Join(allErrors, err)
	}

	return allErrors
}
