package bgp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/microcluster/v2/state"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/ovn/paths"
	"github.com/canonical/microovn/microovn/snap"
	"github.com/zitadel/logging"
)

// BirdService - Name of the Bird routing daemon service managed by MicroOVN
const BirdService = "bird"

// birdTemplateInput - input data for the birdConfTemplate
type birdTemplateInput struct {
	VrfTableID     string
	VrfName        string
	RouterID       string
	ExtConnections []types.BgpExternalConnection
	ASN            string
}

// birdConfTemplate - a template of a Bird configuration file that enables BGP daemon in dynamic
// mode on specified interfaces.
var birdConfTemplate = template.Must(
	template.New("bird.conf").
		Funcs(template.FuncMap{"ifaceName": getBgpRedirectIfaceName}).
		Parse(`
log syslog all;
protocol device {};
protocol direct {
	disabled;	# Disable learning directly connected routes
	ipv4;
	ipv6;
}
protocol kernel {
	ipv4 {
		export all;
	};
	learn;
	kernel table {{ .VrfTableID }};
}

protocol static {
	ipv4;
}

filter no_default_v4 {
	if net = 0.0.0.0/0 then reject;
	accept;
}

filter no_default_v6 {
	if net = ::/0 then reject;
	accept;
}
{{ range .ExtConnections }}
protocol bgp microovn_{{ .Iface }} {
	router id {{ $.RouterID }};
	interface "{{ ifaceName .Iface }}";
	vrf "{{ $.VrfName }}";
	local as {{ $.ASN }};
	neighbor range fe80::/10 external;
	dynamic name "dyn_microovn_{{ .Iface }}_";
	ipv4 {
		next hop self ebgp;
		extended next hop on;
		require extended next hop on;
		import all;
		export filter no_default_v4;
	};
	ipv6 {
		import all;
		export filter no_default_v6;
	};
}
{{ end }}
`))

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

	err := snap.Start(BirdService, true)
	if err != nil {
		logging.Errorf("Failed to start %s service: %s", BirdService, err)
		err = errors.New("failed to start BGP service")
		return errors.Join(err, DisableService(ctx, s))
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
		return errors.Join(err, DisableService(ctx, s))
	}

	err = createExternalNetworks(ctx, s, extConnections)
	if err != nil {
		return errors.Join(err, DisableService(ctx, s))
	}

	err = createVrf(ctx, s, extConnections, extraConfig.Vrf)
	if err != nil {
		return errors.Join(err, DisableService(ctx, s))
	}

	err = redirectBgp(ctx, s, extConnections, extraConfig.Vrf)
	if err != nil {
		return errors.Join(err, DisableService(ctx, s))
	}

	if extraConfig.Asn != "" {
		err = configureBirdBgp(ctx, s, extConnections, extraConfig.Vrf, extraConfig.Asn)
		if err != nil {
			return errors.Join(err, DisableService(ctx, s))
		}
	}

	return nil
}

// DisableService stops and disables BGP services managed by MicroOVN.
func DisableService(ctx context.Context, s state.State) error {
	var allErrors error

	err := snap.Stop(BirdService, true)
	if err != nil {
		logging.Warnf("Failed to stop %s service: %s", BirdService, err)
		allErrors = errors.Join(allErrors, errors.New("failed to stop BGP service"))
	}

	err = teardownAll(ctx, s)
	if err != nil {
		allErrors = errors.Join(allErrors, err)
	}

	return allErrors
}

// configureBirdBgp configures the Bird Routing Daemon to start BGP processes listening on each interface in
// extConnections.
// Each BGP daemon is connected to the VRF table specified by "tableID". It will announce routes from the VRF
// to its peers, and it will insert routes announced by its peers into the same VRF.
// All BGP daemons will be configured with the provided local ASN.
func configureBirdBgp(ctx context.Context, s state.State, extConnections []types.BgpExternalConnection, tableID string, asn string) error {
	vrfName := getVrfName(tableID)

	configFile, err := os.OpenFile(paths.BirdConfigFile(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open Bird configuration file for writing: %w", err)
	}

	err = birdConfTemplate.Execute(configFile, birdTemplateInput{
		VrfTableID:     tableID,
		VrfName:        vrfName,
		RouterID:       generateBGPRouterID(getLrpName(s, extConnections[0].Iface)),
		ExtConnections: extConnections,
		ASN:            asn,
	})
	if err != nil {
		return fmt.Errorf("failed to render Bird configuration template: %w", err)
	}

	out, err := shared.RunCommandContext(ctx, filepath.Join(paths.Wrappers(), "birdc"), "configure")
	if err != nil {
		return fmt.Errorf("failed to apply Bird configuration: %w: %s", err, out)
	}
	return err
}
