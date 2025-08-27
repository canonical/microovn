package bgp

import (
	"bufio"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/state"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/netplan"
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

// generateBGPRouterID returns a router-id address based on a string. The returned
// router-id will always be same for given interface name.
// Warning: There is no guarantee that the address won't conflict with other
// router-ids present in the AS.
func generateBGPRouterID(s string) string {
	routerID := ""
	hash := md5.Sum([]byte(s))
	for i := 0; i < 4; i++ {
		routerID += fmt.Sprintf("%d.", hash[i])
	}
	return strings.TrimRight(routerID, ".")
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

func getBgpVethName(externalIface string) string {
	return fmt.Sprintf("v%s", externalIface)
}

// getBgpRedirectIfaceName returns name of the system interface to which all BGP
// traffic from externalIface network is redirected.
func getBgpRedirectIfaceName(externalIface string) string {
	return fmt.Sprintf("%s-bgp", getBgpVethName(externalIface))
}

// getBgpRedirectIfacePeerName returns name of the peer to the bgp iface
func getBgpRedirectIfacePeerName(externalIface string) string {
	return fmt.Sprintf("%s-brg", getBgpVethName(externalIface))
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

		_, err = ovnCmd.NBCtlCluster(ctx,
			"--",
			// Create Logical Router Port
			"lrp-add", lrName, lrpName, lrpMac,
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
		"set", "Logical_Router", lrName,
		"options:dynamic-routing=true",
		fmt.Sprintf("options:requested-tnl-key=%s", tableID),
	)
	if err != nil {
		return fmt.Errorf("failed to create vrf for LR '%s': %v", lrName, err)
	}

	for _, extConnection := range extConnections {
		lrpName := getLrpName(s, extConnection.Iface)
		_, err = ovnCmd.NBCtlCluster(ctx,
			"lrp-set-options", lrpName, "dynamic-routing-maintain-vrf=true", "dynamic-routing-redistribute=nat,lb",
		)
		if err != nil {
			return fmt.Errorf("failed to enable vrf for LRP '%s': %v", lrpName, err)
		}
	}
	return nil
}

func generateVeth(ctx context.Context, s state.State, extConnections []types.BgpExternalConnection, tableID string) error {

	vrfName := getVrfName(tableID)

	brInt, err := getOvnIntegrationBridge(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to lookup integration bridge: %v", err)
	}

	np := netplan.NewConfig()
	brIntInterfaces := []string{}
	vrfInterfaces := []string{}

	for _, extConnection := range extConnections {
		bgpInterface := getBgpRedirectIfaceName(extConnection.Iface)
		brgInterface := getBgpRedirectIfacePeerName(extConnection.Iface)
		mac := generateLrpMac(getLrpName(s, extConnection.Iface))

		// Add to virtual ethernet
		np.AddVeth(bgpInterface, brgInterface, mac)
		np.AddVeth(brgInterface, bgpInterface, "")
		brIntInterfaces = append(brIntInterfaces, brgInterface)
		vrfInterfaces = append(vrfInterfaces, bgpInterface)
	}

	np.AddVRF(vrfName, tableID, vrfInterfaces)
	np.AddBridge(brInt, brIntInterfaces)

	filename := "90-microovn-bgp-veth.yaml"
	err = netplan.WriteToNetplan(ctx, filename, *np)
	if err != nil {
		return err
	}

	return netplan.Apply(ctx)
}

// redirectBgp creates a port in OVS, moves it to the VRF specified by "tableID" and configures OVN to redirect
// BGP+BFD traffic from the associated Logical Router Ports to the newly created ports.
func redirectBgp(ctx context.Context, s state.State, extConnections []types.BgpExternalConnection, tableID string) error {
	vrfName := getVrfName(tableID)

	err := generateVeth(ctx, s, extConnections, tableID)
	if err != nil {
		return err
	}

	for _, extConnection := range extConnections {
		lsName := getLsName(s, extConnection.Iface)
		lrpName := getLrpName(s, extConnection.Iface)
		brgIface := getBgpRedirectIfacePeerName(extConnection.Iface)
		bgpIface := getBgpRedirectIfaceName(extConnection.Iface)
		bgpLsp := fmt.Sprintf("lsp-%s-%s", s.Name(), bgpIface)

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
			"--",
			"set", "Logical_Router_Port", lrpName, "ipv6_ra_configs:send_periodic=true",
			"--",
			"set", "Logical_Router_Port", lrpName, "ipv6_ra_configs:address_mode=slaac",
			"--",
			"set", "Logical_Router_Port", lrpName, "ipv6_ra_configs:max_interval=1",
			"--",
			"set", "Logical_Router_Port", lrpName, "ipv6_ra_configs:min_interval=1",
		)
		if err != nil {
			return fmt.Errorf("failed to create LSP for BGP redirect '%s': %v", bgpLsp, err)
		}

		// Create OVS port and associate it with the LSP
		_, err = ovnCmd.VSCtl(ctx, s,
			"--",
			"set", "Port", brgIface, fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
			fmt.Sprintf("external-ids:%s=%s", BgpVrfTable, vrfName),

			"--",
			"set", "Interface", brgIface, "type=system", fmt.Sprintf("external_ids:iface-id=%s", bgpLsp),
			fmt.Sprintf("external-ids:%s=true", BgpManagedTag),
		)
		if err != nil {
			return fmt.Errorf("failed to create port for BGP redirect '%s': %v", brgIface, err)
		}
	}
	return nil
}

// startBgpUnnumbered configures BGP process for each external connection. A BGP daemon is started on each interface
// in extConnections, using provided ASN and configured to use "BGP Unnumbered" (auto-discovery mechanism).
// Resulting running configuration is then saved to the startup config.
func startBgpUnnumbered(_ context.Context, s state.State, extConnections []types.BgpExternalConnection, tableID string, asn string) error {
	vrfName := getVrfName(tableID)

	var confBuilder strings.Builder
	fmt.Fprintln(&confBuilder, "configure")

	// Ensure we don't announce any default route from VRF to our peer.
	fmt.Fprint(&confBuilder, `
ip prefix-list no-default seq 5 deny 0.0.0.0/0
ip prefix-list no-default seq 10 permit 0.0.0.0/0 le 32
ipv6 prefix-list no-default seq 5 deny ::/0
ipv6 prefix-list no-default seq 10 permit ::/0 le 128
`)
	fmt.Fprintf(&confBuilder, "router bgp %s vrf %s\n", asn, vrfName)
	for _, connection := range extConnections {
		var ifaceUsed = getBgpRedirectIfaceName(connection.Iface)
		routerID := generateBGPRouterID(getLrpName(s, connection.Iface))
		fmt.Fprintf(&confBuilder, "bgp router-id %s\n", routerID)
		fmt.Fprintf(&confBuilder,
			"neighbor %s interface remote-as external\n",
			ifaceUsed,
		)

		// Redistribute IPv4 routes announced by OVN.
		fmt.Fprint(&confBuilder,
			"address-family ipv4 unicast\n",
		)
		fmt.Fprint(&confBuilder, "redistribute kernel\n")
		fmt.Fprintf(&confBuilder,
			"neighbor %s prefix-list no-default out\n",
			ifaceUsed,
		)
		fmt.Fprintln(&confBuilder,
			"exit-address-family",
		)

		// Enable IPv6 address family.
		fmt.Fprint(&confBuilder,
			"address-family ipv6 unicast\n",
		)
		fmt.Fprintf(&confBuilder,
			"neighbor %s soft-reconfiguration inbound\n",
			ifaceUsed,
		)
		fmt.Fprintf(&confBuilder,
			"neighbor %s prefix-list no-default out\n",
			ifaceUsed,
		)
		fmt.Fprintln(&confBuilder,
			"redistribute kernel",
		)
		fmt.Fprintf(&confBuilder,
			"neighbor %s activate\n",
			ifaceUsed,
		)
		fmt.Fprintln(&confBuilder,
			"exit-address-family",
		)
	}
	fmt.Fprintln(&confBuilder, "do copy running-config startup-config")

	cmd := exec.Command(filepath.Join(paths.Wrappers(), "vtysh"))
	cmd.Stdin = strings.NewReader(confBuilder.String())
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	return err
}

func moveInterfaceToVrf(ctx context.Context, iface string, vrf string) error {
	// Move the port to the VRF, set its IP and MAC address, and bring it UP
	_, err := shared.RunCommandContext(ctx, "ip", "link", "set", "dev", iface, "master", vrf)
	if err != nil {
		return fmt.Errorf("failed to move interface '%s' to VRF '%s': %v", iface, vrf, err)
	}

	_, err = shared.RunCommandContext(ctx, "ip", "link", "set", "dev", iface, "up")
	if err != nil {
		return fmt.Errorf("failed bring interface '%s' UP: %v", iface, err)
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

	err = netplan.Cleanup(ctx, "90-microovn-bgp-veth.yaml")
	if err != nil {
		allErrors = errors.Join(allErrors, fmt.Errorf("failed to cleanup netplan: %v", err))
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
