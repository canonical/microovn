package bgp

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/state"
	"github.com/canonical/microovn/microovn/api/types"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
)

// getOvnIntegrationBridge returns current value of "external-ids:ovn-bridge" from
// the Open_vSwitch table in the OVS database. It returns default value 'br-int' if the
// key does not exist in external-ids.
func getOvnIntegrationBridge(ctx context.Context, s state.State) (string, error) {
	brName, err := vsctlGetIfExists(ctx, s, "Open_vSwitch", ".", "external-ids", "ovn-bridge")
	if brName == "" {
		brName = "br-int"
	}
	return brName, err
}

// getPhysnetName returns physical network name that can be used for setting "ovn-bridge-mappings"
// in OVS and "network_name" for Logical Switch Ports to get connectivity to external network via
// specific interface. This name is unique and consistent for each host and interface.
func getPhysnetName(s state.State, interfaceName string) string {
	return fmt.Sprintf("physnet_%s_%s", s.Name(), interfaceName)
}

// getLrName returns name for the Logical Router to be used for BGP redirecting. This name is
// unique and consistent for each host.
func getLrName(s state.State) string {
	return fmt.Sprintf("lr-%s-microovn", s.Name())
}

// getLsName returns name for the of the Logical Switch that should be used for connection with external
// network. Argument "iface" is the name of the physical interface that provides this connectivity. This
// name is unique for each host and interface.
func getLsName(s state.State, iface string) string {
	return fmt.Sprintf("ls-%s-%s", s.Name(), iface)
}

// getLrpName returns name of the Logical Router Port that should be connected to the Logical Switch that
// provides connectivity to external network. Argument "iface" is a name of a physical interface that provides
// this connectivity. This name is unique and consistent for each host and interface.
func getLrpName(s state.State, iface string) string {
	return fmt.Sprintf("lrp-%s-%s", s.Name(), iface)
}

// getExternalConnectionCidr returns CIDR IPv4 notation for the IPv4 address and network mask defined in
// types.BgpExternalConnection.
// Example:
//
//	types.BgpExternalConnection.IPAddress: 192.0.2.1
//	types.BgpExternalConnection.IPMask:    255.255.255.0
//
// Result: "192.0.2.1/24"
func getExternalConnectionCidr(extConn types.BgpExternalConnection) string {
	lrpIP4Mask, _ := extConn.IPMask.Size()
	return fmt.Sprintf("%s/%d", extConn.IPAddress, lrpIP4Mask)
}

// generateLrpMac returns a local unicast MAC address based on an interface name. The returned
// address will always be same for given interface name.
// Warning: There is no guarantee that the address won't conflict with other MAC addresses
// present in the network.
func generateLrpMac(ifaceName string) string {
	macAddr := "02:"
	nameHash := md5.Sum([]byte(ifaceName))
	for i := 0; i < 5; i++ {
		macAddr += fmt.Sprintf("%02x:", nameHash[i])
	}
	return strings.TrimRight(macAddr, ":")
}

// vsctlGetIfExists runs 'ovs-vsctl' get to retrieve record [column [key]] from the
// specified table. Returned string has whitespace and quotations trimmed.
// If the 'ovs-vsctl' command failed due to the "key" not being found in "column",
// this function returns empty string without error.
func vsctlGetIfExists(ctx context.Context, s state.State, table string, record string, column string, key string) (string, error) {
	args := []string{"get", table, record}
	if column != "" {
		if key != "" {
			column = fmt.Sprintf("%s:%s", column, key)
		}
		args = append(args, column)
	}
	result, err := ovnCmd.VSCtl(ctx, s, args...)
	// Don't return error if command failed due to
	if err != nil && !strings.Contains(fmt.Sprintf("%v", err), "ovs-vsctl: no key") {
		return "", err
	}
	return strings.Trim(strings.TrimSpace(result), "\""), nil
}

// createExternalBridges sets up OVS bridge for each external connection defined in "extConnections" argument.
// Physical interface defined in the external connection will be plugged to this bridge and the bridge will
// be named "<iface>-br". Additionally, a physical network name will be constructed with getPhysnetName() and
// will be added to "ovn-bridge-mappings" in the OVS database.
func createExternalBridges(ctx context.Context, s state.State, extConnections []types.BgpExternalConnection) error {
	for _, extConnection := range extConnections {
		bridgeName := fmt.Sprintf("br-%s", extConnection.Iface)
		physnet := getPhysnetName(s, extConnection.Iface)
		bridgeMap, err := vsctlGetIfExists(ctx, s, "Open_vSwitch", ".", "external-ids", "ovn-bridge-mappings")
		if err != nil {
			return fmt.Errorf("failed to lookup ovn-bridge-mappings: %v", err)
		}
		if bridgeMap == "" {
			bridgeMap = fmt.Sprintf("%s:%s", physnet, bridgeName)
		} else {
			bridgeMap = fmt.Sprintf("%s,%s:%s", bridgeMap, physnet, bridgeName)

		}

		_, err = ovnCmd.VSCtl(ctx, s,
			"--",
			"add-br", bridgeName,
			"--",
			"set", "Open_vSwitch", ".", fmt.Sprintf("external-ids:ovn-bridge-mappings=\"%s\"", bridgeMap),
			"--",
			"add-port", bridgeName, extConnection.Iface,
		)
		if err != nil {
			logger.Errorf("failed to create external bridge for interface '%s': %v", extConnection.Iface, err)
			return err
		}
	}
	return nil
}

// createExternalNetworks creates a single Logical Router and connects it to each external network defined
// in "extConnections" argument. The connection is facilitated via Logical switches, each external network
// is represented by its own switch.
func createExternalNetworks(ctx context.Context, s state.State, extConnections []types.BgpExternalConnection) error {
	// Create Logical Router
	lrName := getLrName(s)
	_, err := ovnCmd.NBCtlCluster(ctx,
		"--",
		"lr-add", lrName,
		"--",
		"set", "Logical_Router", lrName, fmt.Sprintf("options:chassis=%s", s.Name()),
	)
	if err != nil {
		logger.Errorf("Failed to create OVN Logical Router for external connectivity: %v", err)
		return err
	}
	for _, extConnection := range extConnections {
		lsName := getLsName(s, extConnection.Iface)
		lspName := fmt.Sprintf("lsp-%s-%s", s.Name(), extConnection.Iface)

		patchName := fmt.Sprintf("patch-%s-%s", s.Name(), extConnection.Iface)
		physnetName := getPhysnetName(s, extConnection.Iface)

		lrpName := getLrpName(s, extConnection.Iface)
		lrpMac := generateLrpMac(lrpName)
		lrpIP4 := getExternalConnectionCidr(extConnection)

		_, err = ovnCmd.NBCtlCluster(ctx,
			"--",
			// Create Logical Router Port
			"lrp-add", lrName, lrpName, lrpMac, lrpIP4,
			"--",
			// Create Logical Switch and connect it to the Logical Router Port
			"ls-add", lsName,
			"--",
			"lsp-add", lsName, lspName,
			"--",
			"lsp-set-type", lspName, "router",
			"--",
			"lsp-set-options", lspName, fmt.Sprintf("router-port=%s", lrpName),
			"--",
			"lsp-set-addresses", lspName, "router",
			"--",
			// Connect Logical Switch with the external network
			"lsp-add", lsName, patchName,
			"--",
			"lsp-set-addresses", patchName, "unknown",
			"--",
			"lsp-set-type", patchName, "localnet",
			"--",
			"lsp-set-options", patchName, fmt.Sprintf("network_name=%s", physnetName),
		)

		if err != nil {
			logger.Errorf("failed to create external networks for interface '%s': %v", extConnection.Iface, err)
			return err
		}
	}
	return nil
}

// createVrf instructs OVN to set up VRF to redistribute NAT and Load Balancer addresses for each Logical Router Port
// that's associated with external connections defined in "extConnections" argument. Only one VRF is set up with table
// ID specified by "tableID" argument. All LRPs redistribute their addresses to this VRF.
func createVrf(ctx context.Context, s state.State, extConnections []types.BgpExternalConnection, tableID string) error {
	lrName := getLrName(s)

	_, err := ovnCmd.NBCtlCluster(ctx,
		"set", "Logical_Router", lrName, fmt.Sprintf("options:requested-tnl-key=%s", tableID),
	)
	if err != nil {
		return fmt.Errorf("failed to create vrf for LR '%s': %v", lrName, err)
	}

	for _, extConnection := range extConnections {
		lrpName := getLrpName(s, extConnection.Iface)
		_, err = ovnCmd.NBCtlCluster(ctx,
			"lrp-set-options", lrpName, "maintain-vrf=true", "redistribute-nat=true", "redistribute-lb-vips=true",
		)
		if err != nil {
			return fmt.Errorf("failed to enable vrf for LRP '%s': %v", lrpName, err)
		}
	}
	return nil
}

// redirectBgp creates a port in OVS, moves it to the VRF specified by "tableID" and configures OVN to redirect
// BGP+BFD traffic from the associated Logical Router Ports to the newly created ports.
func redirectBgp(ctx context.Context, s state.State, extConnections []types.BgpExternalConnection, tableID string) error {
	intBr, err := getOvnIntegrationBridge(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to lookup integration bridge: %v", err)
	}
	vrfName := fmt.Sprintf("ovnvrf%s", tableID)

	for _, extConnection := range extConnections {
		lsName := getLsName(s, extConnection.Iface)
		lrpName := getLrpName(s, extConnection.Iface)
		bgpIface := fmt.Sprintf("%s-bgp", extConnection.Iface)
		bgpLsp := fmt.Sprintf("lsp-%s-%s-bgp", s.Name(), extConnection.Iface)
		mac := generateLrpMac(lrpName)
		bgpIfaceIp4 := getExternalConnectionCidr(extConnection)

		// Create Logical Switch Port to which the BGP+BFD traffic will be redirected
		_, err := ovnCmd.NBCtlCluster(ctx,
			"--",
			"lsp-add", lsName, bgpLsp,
			"--",
			"lsp-set-addresses", bgpLsp, "unknown",
			"--",
			"add", "Logical_Router_Port", lrpName, "options", fmt.Sprintf("routing-protocol-redirect=%s", bgpLsp),
			"--",
			"add", "Logical_Router_Port", lrpName, "options", "routing-protocols=\"BGP,BFD\"",
		)
		if err != nil {
			return fmt.Errorf("failed to create LSP for BGP redirect '%s': %v", bgpLsp, err)
		}

		// Create OVS port and associate it with the LSP
		_, err = ovnCmd.VSCtl(ctx, s,
			"--",
			"add-port", intBr, bgpIface,
			"--",
			"set", "Interface", bgpIface, "type=internal", fmt.Sprintf("external_ids:iface-id=%s", bgpLsp),
			fmt.Sprintf("mac=\"%s\"", mac),
		)
		if err != nil {
			return fmt.Errorf("failed to create port for BGP redirect '%s': %v", bgpIface, err)
		}

		// Move the port to the VRF, set its IP and MAC address, and bring it UP
		_, err = shared.RunCommandContext(ctx, "ip", "link", "set", "dev", bgpIface, "master", vrfName)
		if err != nil {
			return fmt.Errorf("failed to move interface '%s' to VRF '%s': %v", bgpIface, vrfName, err)
		}

		_, err = shared.RunCommandContext(ctx, "ip", "link", "set", "dev", bgpIface, "up")
		if err != nil {
			return fmt.Errorf("failed bring interface '%s' UP: %v", bgpIface, err)
		}

		_, err = shared.RunCommandContext(ctx, "ip", "address", "add", bgpIfaceIp4, "dev", bgpIface)
		if err != nil {
			return fmt.Errorf("failed to set IPv4 on interface '%s': %v", bgpIface, err)
		}
	}
	return nil
}
