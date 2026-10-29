package bgpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/canonical/lxd/shared/logger"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/bgp"
	microOvnClient "github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/config"
	microOvnNode "github.com/canonical/microovn/microovn/node"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
)

var BgpSetupEndpoint = microTypes.Endpoint{
	Path: "bgp/setup",
	Post: microTypes.EndpointAction{Handler: setupBgpHandler, AllowUntrusted: false, ProxyTarget: true},
}

var BgpCheckInterfaceEndpoint = microTypes.Endpoint{
	Path: "bgp/check-interface",
	Get:  microTypes.EndpointAction{Handler: checkInterfaceHandler, AllowUntrusted: false, ProxyTarget: true},
}

func setupBgpHandler(s microTypes.State, r *http.Request) microTypes.Response {
	resp := types.SetupBgpResponse{}

	if !microTypes.IsNotification(r) {
		var req types.SetupBgpRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			return microTypes.BadRequest(errors.New("failed to decode request body"))
		}

		if err := req.Validate(); err != nil {
			return microTypes.BadRequest(err)
		}

		ifaceName := req.ExternalConnection
		if req.Bridge != "" {
			ifaceName = req.Bridge
		}

		err = config.SetConfig(r.Context(), s, types.BgpConfigKeyCluster, "true")
		if err != nil {
			return microTypes.InternalError(fmt.Errorf("failed to save BGP cluster config: %w", err))
		}

		if req.ExternalConnection != "" {
			err = config.SetConfig(r.Context(), s, types.BgpConfigKeyInterface, req.ExternalConnection)
		} else {
			err = config.SetConfig(r.Context(), s, types.BgpConfigKeyBridge, req.Bridge)
		}
		if err != nil {
			return microTypes.InternalError(fmt.Errorf("failed to save BGP interface/bridge config: %w", err))
		}

		if req.Asn != "" {
			err = config.SetConfig(r.Context(), s, types.BgpConfigKeyAsn, req.Asn)
			if err != nil {
				return microTypes.InternalError(fmt.Errorf("failed to save BGP ASN config: %w", err))
			}
		}
		if req.AsnRange[0] != 0 || req.AsnRange[1] != 0 {
			err = config.SetConfig(r.Context(), s, types.BgpConfigKeyAsnRange,
				fmt.Sprintf("%d-%d", req.AsnRange[0], req.AsnRange[1]))
			if err != nil {
				return microTypes.InternalError(fmt.Errorf("failed to save BGP ASN range config: %w", err))
			}
		}
		if req.ConnectionIP4Range != "" {
			err = config.SetConfig(r.Context(), s, types.BgpConfigKeyConnectionIP4Range, req.ConnectionIP4Range)
			if err != nil {
				return microTypes.InternalError(fmt.Errorf("failed to save BGP connection IP range config: %w", err))
			}
		}

		cluster, err := s.Connect().Cluster(true)
		if err != nil {
			logger.Errorf("Failed to get a client for every cluster member: %v", err)
			return microTypes.InternalError(fmt.Errorf("failed to get cluster clients: %w", err))
		}

		var interfaceErrors []string
		err = cluster.Query(r.Context(), false, func(ctx context.Context, c microTypes.Client) error {
			var result map[string]bool
			checkURL := &url.URL{Path: "bgp/check-interface", RawQuery: fmt.Sprintf("name=%s", ifaceName)}
			err := c.Query(ctx, "GET", types.APIVersion, checkURL, nil, &result)
			if err != nil {
				interfaceErrors = append(interfaceErrors, fmt.Sprintf("%s: failed to check interface: %v", c.URL().String(), err))
				return nil
			}
			if exists, ok := result["exists"]; !ok || !exists {
				interfaceErrors = append(interfaceErrors, fmt.Sprintf("%s: interface/bridge '%s' not found", c.URL().String(), ifaceName))
			}
			return nil
		})
		if err != nil {
			return microTypes.SmartError(err)
		}

		if len(interfaceErrors) > 0 {
			resp.Success = false
			resp.Message = "Interface/bridge check failed on some nodes"
			resp.Errors = interfaceErrors
			return microTypes.SyncResponse(false, &resp)
		}

		err = cluster.Query(r.Context(), true, func(ctx context.Context, c microTypes.Client) error {
			clientURL := c.URL()
			logger.Infof("Requesting cluster member at '%s' to set up BGP", clientURL.String())

			setupReq := types.SetupBgpRequest{}
			result, err := microOvnClient.SetupBgp(ctx, c, setupReq)
			if err != nil {
				errMsg := fmt.Sprintf("node %s: %v", clientURL.String(), err)
				resp.Errors = append(resp.Errors, errMsg)
			} else if !result.Success {
				for _, e := range result.Errors {
					resp.Errors = append(resp.Errors, fmt.Sprintf("node %s: %s", clientURL.String(), e))
				}
			}
			return nil
		})
		if err != nil {
			return microTypes.SmartError(err)
		}
	}

	err := EnableBgpOnNodeFromClusterConfig(r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to set up BGP on this node: %v", err)
		resp.Errors = append(resp.Errors, fmt.Sprintf("node %s: %v", s.Name(), err))
	}

	if len(resp.Errors) > 0 {
		resp.Success = false
		resp.Message = fmt.Sprintf("BGP setup completed with %d error(s)", len(resp.Errors))
	} else {
		resp.Success = true
		resp.Message = "BGP setup completed on all nodes"
	}

	return microTypes.SyncResponse(true, &resp)
}

func checkInterfaceHandler(s microTypes.State, r *http.Request) microTypes.Response {
	ifaceName := r.URL.Query().Get("name")
	if ifaceName == "" {
		return microTypes.BadRequest(errors.New("missing 'name' query parameter"))
	}

	exists := interfaceExists(ifaceName)

	return microTypes.SyncResponse(true, map[string]bool{"exists": exists})
}

func interfaceExists(name string) bool {
	_, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", name))
	return err == nil
}

func CheckBgpInterfaceFromClusterConfig(ctx context.Context, s microTypes.State) error {
	ifaceItem, _ := config.GetConfig(ctx, s, types.BgpConfigKeyInterface)
	if ifaceItem != nil && ifaceItem.Value != "" {
		if !interfaceExists(ifaceItem.Value) {
			return fmt.Errorf("required BGP interface '%s' not found on this node", ifaceItem.Value)
		}
		return nil
	}

	brItem, _ := config.GetConfig(ctx, s, types.BgpConfigKeyBridge)
	if brItem != nil && brItem.Value != "" {
		if !interfaceExists(brItem.Value) {
			return fmt.Errorf("required BGP bridge '%s' not found on this node", brItem.Value)
		}
		return nil
	}

	return fmt.Errorf("no BGP interface or bridge configured in cluster setup")
}

func EnableBgpOnNodeFromClusterConfig(ctx context.Context, s microTypes.State) error {
	bgpConfig := &types.ExtraBgpConfig{}

	ifaceItem, err := config.GetConfig(ctx, s, types.BgpConfigKeyInterface)
	if err != nil {
		return fmt.Errorf("failed to read BGP interface config: %w", err)
	}
	if ifaceItem != nil {
		bgpConfig.ExternalConnection = ifaceItem.Value
	}

	brItem, err := config.GetConfig(ctx, s, types.BgpConfigKeyBridge)
	if err != nil {
		return fmt.Errorf("failed to read BGP bridge config: %w", err)
	}
	if brItem != nil {
		bgpConfig.Bridge = brItem.Value
	}

	asnItem, err := config.GetConfig(ctx, s, types.BgpConfigKeyAsn)
	if err != nil {
		logger.Errorf("failed to read BGP ASN config: %v", err)
	}
	if asnItem != nil {
		bgpConfig.Asn = asnItem.Value
	}

	asnRangeItem, err := config.GetConfig(ctx, s, types.BgpConfigKeyAsnRange)
	if err != nil {
		logger.Errorf("failed to read BGP ASN range config: %v", err)
	}
	if asnRangeItem != nil {
		rawConfig := map[string]string{
			"asn_range": asnRangeItem.Value,
		}
		err := bgpConfig.FromMap(rawConfig)
		if err != nil {
			return fmt.Errorf("failed to parse ASN range from cluster config: %w", err)
		}
	}

	extraConfig := &types.ExtraServiceConfig{
		BgpConfig: bgpConfig,
	}

	return microOvnNode.EnableService(ctx, s, types.SrvBgp, extraConfig)
}

const ovsBgpEnabledKey = "microovn-bgp-enabled"
const ovsBgpVrfKey = "microovn-bgp-vrf"
const ovsBgpAsnKey = "microovn-bgp-asn"
const ovsBgpAsnRangeKey = "microovn-bgp-asn-range"

func EnableBgpFromOVSExternalIDs(ctx context.Context, s microTypes.State) error {
	switchActive, err := microOvnNode.HasServiceActive(ctx, s, types.SrvSwitch)
	if err != nil {
		return fmt.Errorf("failed to check switch service status: %w", err)
	}
	if !switchActive {
		return nil
	}

	bridgeNames, err := findBridgesByExternalID(ctx, s, ovsBgpEnabledKey, "true")
	if err != nil {
		return fmt.Errorf("failed to look up BGP-enabled OVS bridges: %w", err)
	}

	if len(bridgeNames) == 0 {
		return nil
	}

	for _, bridgeName := range bridgeNames {
		logger.Infof("Enabling BGP on bridge %s from OVS external IDs", bridgeName)

		bgpConfig := &types.ExtraBgpConfig{
			Bridge: bridgeName,
		}

		vrf, err := bgp.VsctlGetIfExists(ctx, s, "Bridge", bridgeName, "external-ids", ovsBgpVrfKey)
		if err != nil {
			return fmt.Errorf("failed to read %s on bridge %s: %w", ovsBgpVrfKey, bridgeName, err)
		}
		if vrf != "" {
			bgpConfig.Vrf = vrf
		}

		asn, err := bgp.VsctlGetIfExists(ctx, s, "Bridge", bridgeName, "external-ids", ovsBgpAsnKey)
		if err != nil {
			return fmt.Errorf("failed to read %s on bridge %s: %w", ovsBgpAsnKey, bridgeName, err)
		}
		if asn != "" {
			bgpConfig.Asn = asn
		} else {
			asnRange, err := bgp.VsctlGetIfExists(ctx, s, "Bridge", bridgeName, "external-ids", ovsBgpAsnRangeKey)
			if err != nil {
				return fmt.Errorf("failed to read %s on bridge %s: %w", ovsBgpAsnRangeKey, bridgeName, err)
			}
			if asnRange != "" {
				parsed, err := types.ParseAsnRange(asnRange)
				if err != nil {
					return fmt.Errorf("invalid %s on bridge %s: %w", ovsBgpAsnRangeKey, bridgeName, err)
				}
				bgpConfig.AsnRange = parsed
			}
		}

		extraConfig := &types.ExtraServiceConfig{
			BgpConfig: bgpConfig,
		}

		err = microOvnNode.EnableService(ctx, s, types.SrvBgp, extraConfig)
		if err != nil {
			return fmt.Errorf("failed to enable BGP from OVS external IDs on bridge %s: %w", bridgeName, err)
		}
	}

	return nil
}

func findBridgesByExternalID(ctx context.Context, s microTypes.State, key, value string) ([]string, error) {
	output, err := ovnCmd.VSCtl(ctx, s, "--bare", "--columns", "name",
		"find", "bridge", fmt.Sprintf("external-ids:%s=%s", key, value),
	)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}
