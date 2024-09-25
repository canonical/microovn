package bgp

import (
	"bufio"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/state"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/frr/vtysh"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/canonical/microovn/microovn/ovn/paths"
)

// BgpManagedTag - a key used in "external_ids" table of various OVN/OVS
// resources to identify those that are created and managed by MicroOVN
// for the purpose of BGP integration
const BgpManagedTag = "microovn-bgp-managed"

// BgpVrfTable - a key used in "external_ids" table of OVS ports used for
// BGP redirecting. It keeps information about which VRF table should these
// ports be assigned
const BgpVrfTable = "microovn-bgp-vrf"

// BgpIfaceIP - a key used in "external_ids" table of OVS ports used for
// BGP redirecting. It keeps track of the IPv4 address that should be
// assigned to the port, to match IPv4 address of the Logical Router Port
// from which the traffic is redirected
const BgpIfaceIP = "microovn-bgp-ip"

// BgpBridgeMapping - a key used in "external_ids" of Open_vSwitch table, to
// keep track of "ovn-bridge-mappings" managed by MicroOVN.
const BgpBridgeMapping = "microovn-bgp-bridge-mapping"

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

// getLsNameChassisMatch returns a string that can be used to match
// names of all Logical Switches used for BGP redirecting on local chassis
func getLsNameChassisMatch(s state.State) string {
	dummyIface := "FOO"
	match, _ := strings.CutSuffix(getLsName(s, dummyIface), dummyIface)
	return match
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

// getVrfName Based on the supplied VRF table ID, return name
// of the VRF that would be created by OVN.
//
// When OVN is requested to maintain VRF, it uses established
// pattern to generate VRF name from the VRF table ID. Following
// this patter is currently our only way to relate table IDs to the
// OVN's VRF names.
func getVrfName(tableID string) string {
	return fmt.Sprintf("ovnvrf%s", tableID)
}

// getBgpRedirectIfaceName returns name of the system interface to which all BGP
// traffic from externalIface network is redirected.
func getBgpRedirectIfaceName(externalIface string) string {
	return fmt.Sprintf("%s-bgp", externalIface)
}

// parseOvnFind parses STDOUT string of OVN/OVS "find" commands with "--bare"
// formatting. Returned value is a list of strings with each element containing
// single, non-empty, line of the "find" result.
func parseOvnFind(stdout string) []string {
	var foundValues []string
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		foundValues = append(foundValues, line)
	}
	return foundValues
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
			"set", "bridge", bridgeName, fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
			"--",
			"set", "Open_vSwitch", ".", fmt.Sprintf("external-ids:ovn-bridge-mappings=\"%s\"", bridgeMap),
			fmt.Sprintf("external-ids:%s=\"%s\"", BgpBridgeMapping, bridgeMap),
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
		"--",
		"set", "Logical_Router", lrName, fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
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
			"set", "Logical_Switch", lsName, fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
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
	vrfName := getVrfName(tableID)

	for _, extConnection := range extConnections {
		lsName := getLsName(s, extConnection.Iface)
		lrpName := getLrpName(s, extConnection.Iface)
		bgpIface := getBgpRedirectIfaceName(extConnection.Iface)
		bgpLsp := fmt.Sprintf("lsp-%s-%s-bgp", s.Name(), extConnection.Iface)
		mac := generateLrpMac(lrpName)
		bgpIfaceIP4 := getExternalConnectionCidr(extConnection)

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
			"set", "Port", bgpIface, fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
			fmt.Sprintf("external-ids:%s=%s", BgpVrfTable, vrfName), fmt.Sprintf("external-ids:%s=%s", BgpIfaceIP, bgpIfaceIP4),
			"--",
			"set", "Interface", bgpIface, "type=internal", fmt.Sprintf("external_ids:iface-id=%s", bgpLsp),
			fmt.Sprintf("mac=\"%s\"", mac),
		)
		if err != nil {
			return fmt.Errorf("failed to create port for BGP redirect '%s': %v", bgpIface, err)
		}

		err = moveInterfaceToVrf(ctx, bgpIface, bgpIfaceIP4, vrfName)
		if err != nil {
			return err
		}
	}
	return nil
}

// startBgpUnnumbered configures BGP process for each external connection. A BGP daemon is started on each interface
// in extConnections, using provided ASN and configured to use "BGP Unnumbered" (auto-discovery mechanism).
// Resulting running configuration is then saved to the startup config.
func startBgpUnnumbered(ctx context.Context, extConnections []types.BgpExternalConnection, tableID string, asn string) error {
	vrfName := getVrfName(tableID)

	vtyCommands := vtysh.NewVtyshCommand("configure")
	vtyCommands.Add(fmt.Sprintf("router bgp %s vrf %s", asn, vrfName))
	for _, connection := range extConnections {
		vtyCommands.Add(fmt.Sprintf(
			"neighbor %s interface remote-as internal", getBgpRedirectIfaceName(connection.Iface),
		))
	}
	vtyCommands.Add("do copy running-config startup-config")

	_, err := vtyCommands.Execute(ctx)
	return err
}

func moveInterfaceToVrf(ctx context.Context, iface string, ipv4Cidr string, vrf string) error {
	// Move the port to the VRF, set its IP and MAC address, and bring it UP
	_, err := shared.RunCommandContext(ctx, "ip", "link", "set", "dev", iface, "master", vrf)
	if err != nil {
		return fmt.Errorf("failed to move interface '%s' to VRF '%s': %v", iface, vrf, err)
	}

	_, err = shared.RunCommandContext(ctx, "ip", "link", "set", "dev", iface, "up")
	if err != nil {
		return fmt.Errorf("failed bring interface '%s' UP: %v", iface, err)
	}

	_, err = shared.RunCommandContext(ctx, "ip", "address", "add", ipv4Cidr, "dev", iface)
	if err != nil {
		return fmt.Errorf("failed to set IPv4 on interface '%s': %v", iface, err)
	}
	return nil
}

// teardownAll removes all resources that were created/configured as part of setting up of
// the BGP redirect. This includes:
//   - Logical Router
//   - Logical Switches
//   - OVS external bridges
//   - OVS ports
//   - OVN bridge mappings
//
// Other OVN resources remain untouched.
func teardownAll(ctx context.Context, s state.State) error {
	var allErrors error
	// Find and remove Logical Router used for BGP redirect
	logicalRouter := getLrName(s)
	_, err := ovnCmd.NBCtlCluster(ctx, "lr-del", logicalRouter)
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete Logical Router '%s': %v", logicalRouter, err))
	}

	// Find and remove Logical Switches used to connect to external networks on the local chassis
	logicalSwitches, err := ovnCmd.NBCtlCluster(ctx, "--bare", "--columns", "name",
		"find", "logical_switch", fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
	)
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to lookup Logical Switches managed by MicroOVN: %v", err))
	} else {
		chassisSwitchNamePrefix := getLsNameChassisMatch(s)
		for _, logicalSwitch := range parseOvnFind(logicalSwitches) {
			// Remove only those switches that are related to the local chassis
			if !strings.HasPrefix(logicalSwitch, chassisSwitchNamePrefix) {
				continue
			}
			_, err = ovnCmd.NBCtlCluster(ctx, "ls-del", logicalSwitch)
			if err != nil {
				allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete Logical Switch '%s': %v", logicalSwitch, err))
			}
		}
	}

	// Find and remove external OVS bridges
	bridges, err := ovnCmd.VSCtl(ctx, s, "--bare", "--columns", "name",
		"find", "bridge", fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
	)
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to lookup OVS Bridges managed by MicroOVN: %v", err))
	} else {
		for _, bridge := range parseOvnFind(bridges) {
			_, err = ovnCmd.VSCtl(ctx, s, "del-br", bridge)
			if err != nil {
				allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete OVS Bridge '%s': %v", bridge, err))
			}
		}
	}

	// Find and remove OVS ports used for BGP redirect
	ports, err := ovnCmd.VSCtl(ctx, s, "--bare", "--columns", "name",
		"find", "port", fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
	)
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to lookup OVS Ports managed by MicroOVN: %v", err))
	} else {
		for _, port := range parseOvnFind(ports) {
			_, err = ovnCmd.VSCtl(ctx, s, "del-port", port)
			if err != nil {
				allErrors = errors.Join(allErrors, fmt.Errorf("failed to delete OVS Port '%s': %v", port, err))
			}
		}
	}

	// Cleanup ovn-bridge mappings for external networks
	ovnBridgeMapping, err := vsctlGetIfExists(ctx, s, "Open_vSwitch", ".", "external-ids", "ovn-bridge-mappings")
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to lookup Open_vSwitch ovn-bridge-mappings: %v", err))
		return allErrors
	}

	microOvnBridgeMapping, err := vsctlGetIfExists(ctx, s, "Open_vSwitch", ".", "external-ids", BgpBridgeMapping)
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to lookup OVN bridge mapping managed by MicroOVN: %v", err))
	} else if len(ovnBridgeMapping) != 0 && len(microOvnBridgeMapping) != 0 {
		// Proceed with updating ovn-bridge-mapping only if it's present (along with 'microovn-bgp-bridge-mapping')
		microOvnBridgeMaps := strings.Split(microOvnBridgeMapping, ",")
		ovnBridgeMaps := strings.Split(ovnBridgeMapping, ",")
		var newBridgeMapping string

		// Remove ovn-bridge-mappings entries that were added by MicroOVN
		for _, bridgeMap := range ovnBridgeMaps {
			if !slices.Contains(microOvnBridgeMaps, bridgeMap) {
				newBridgeMapping = fmt.Sprintf("%s,%s", newBridgeMapping, bridgeMap)
			}
		}
		newBridgeMapping = strings.Trim(newBridgeMapping, ",")

		if newBridgeMapping == "" {
			_, err = ovnCmd.VSCtl(ctx, s, "remove", "Open_vSwitch", ".", "external-ids", "ovn-bridge-mappings")
		} else {
			_, err = ovnCmd.VSCtl(ctx, s,
				"set", "Open_vSwitch", ".",
				fmt.Sprintf("external-ids:ovn-bridge-mappings=%s", newBridgeMapping),
			)
		}
		if err != nil {
			allErrors = errors.Join(allErrors, fmt.Errorf(
				"failed to remove MicroOVN managed bridge mappings from ovn-bridge-mappings: %v", err),
			)
		}
	}

	// Remove microovn-bgp-bridge-mapping entirely
	_, err = ovnCmd.VSCtl(ctx, s, "remove", "Open_vSwitch", ".", "external-ids", BgpBridgeMapping)
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to remove %s: %v", BgpBridgeMapping, err))
	}

	// Backup and reset FRR's config
	backupConfig := fmt.Sprintf("%s_%d", paths.FrrStartupConfig(), time.Now().Unix())
	_, err = shared.RunCommandContext(ctx, "cp", paths.FrrStartupConfig(), backupConfig)
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf(
			"failed to backup FRR startup config. Will not proceed with its removal: %v", err),
		)
	} else {
		_, err = shared.RunCommandContext(ctx, "cp", paths.FrrDefaultConfig(), paths.FrrConfigDir())
		if err != nil {
			allErrors = errors.Join(allErrors, fmt.Errorf("failed to reset FRR startup config: %v", err))
		}
	}

	return allErrors
}
