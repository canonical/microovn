package facts

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/canonical/microcluster/v3/state"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/ovn/cmd"
)

func CollectNetworkProbe(
	ctx context.Context,
	s state.State,
) types.InspectionNetworkProbe {
	probe := types.InspectionNetworkProbe{
		Scope: types.InspectionScopeCluster,
	}

	dhcp, collectionErrors := collectDHCP(ctx, s)
	if len(collectionErrors) > 0 {
		probe.CollectionErrors = append(probe.CollectionErrors, collectionErrors...)
	} else {
		probe.DHCPOptions = dhcp
	}

	return probe
}

type dhcpOptionsResponse struct {
	Data []dhcp `json:"data"`
}

type dhcp struct {
	UUID        string
	CIDR        string
	Options     dhcpOptions
	ExternalIDs map[string]string
}

type dhcpOptions struct {
	ClasslessStaticRoutes []string
	DNSServers            []string
	DomainName            string
	LeaseTime             string
	MTU                   string
	Router                string
	ServerID              string
	ServerMAC             string
}

type logicSwitchPortResponse struct {
	Data []logicSwitchPort `json:"data"`
}

type logicSwitchPort struct {
	Name            string
	DHCPOptionsUUID string
}

func parseSet(raw string) []string {
	if raw == "" {
		return nil
	}

	raw = strings.Trim(raw, "{}")
	parts := strings.Split(raw, ",")
	res := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			res = append(res, part)
		}
	}
	return res
}

func parseClasslessStaticRoutes(raw string) ([]string, error) {
	values := parseSet(raw)
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("invalid classless static routes, expected destination/gateway pairs, got %d values", len(values))
	}

	routes := make([]string, 0, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		routes = append(routes, values[i]+","+values[i+1])
	}

	return routes, nil
}

func parseMap(raw json.RawMessage) (map[string]string, error) {
	var tuple []json.RawMessage

	if err := json.Unmarshal(raw, &tuple); err != nil {
		return nil, err
	}
	if len(tuple) != 2 {
		return nil, fmt.Errorf("failed to parse map tuple, expected 2 fields, got %d", len(tuple))
	}

	var typeStr string
	if err := json.Unmarshal(tuple[0], &typeStr); err != nil {
		return nil, fmt.Errorf("failed to parse map type: %w", err)
	}
	if typeStr != "map" {
		return nil, fmt.Errorf("invalid map type, expected 'map', got %q", typeStr)
	}

	var pairs [][]string
	if err := json.Unmarshal(tuple[1], &pairs); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		if len(pair) != 2 {
			return nil, fmt.Errorf("failed to parse map row, expected 2 fields, got %d", len(pair))
		}
		result[pair[0]] = pair[1]
	}

	return result, nil
}

func (s *dhcp) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != 4 {
		return fmt.Errorf("invalid subnet row, expected 4 fields, got %d", len(raw))
	}

	var uuidTuple []string
	if err := json.Unmarshal(raw[0], &uuidTuple); err != nil {
		return fmt.Errorf("failed to parse subnet UUID: %w", err)
	}
	if len(uuidTuple) != 2 || uuidTuple[0] != "uuid" {
		return fmt.Errorf("invalid subnet UUID tuple, expected [uuid, value]")
	}
	s.UUID = uuidTuple[1]

	var cidr string
	if err := json.Unmarshal(raw[1], &cidr); err != nil {
		return fmt.Errorf("failed to parse subnet CIDR: %w", err)

	}
	s.CIDR = cidr

	options, err := parseMap(raw[2])
	if err != nil {
		return fmt.Errorf("failed to parse options map: %w", err)
	}
	classlessStaticRoutes, err := parseClasslessStaticRoutes(options["classless_static_route"])
	if err != nil {
		return err
	}
	s.Options = dhcpOptions{
		ClasslessStaticRoutes: classlessStaticRoutes,
		DNSServers:            parseSet(options["dns_server"]),
		DomainName:            strings.Trim(options["domain_name"], `"`),
		LeaseTime:             options["lease_time"],
		MTU:                   options["mtu"],
		Router:                options["router"],
		ServerID:              options["server_id"],
		ServerMAC:             options["server_mac"],
	}

	externalIDs, err := parseMap(raw[3])
	if err != nil {
		return fmt.Errorf("failed to parse external_ids map: %w", err)
	}
	s.ExternalIDs = externalIDs

	return nil
}

func (s *logicSwitchPort) UnmarshalJSON(b []byte) error {
	var tuple []json.RawMessage
	if err := json.Unmarshal(b, &tuple); err != nil {
		return err
	}
	if len(tuple) != 2 {
		return fmt.Errorf("invalid logic switch port row, expected 2 fields, got %d", len(tuple))
	}

	var name string
	if err := json.Unmarshal(tuple[0], &name); err != nil {
		return fmt.Errorf("failed to parse logic switch port name: %w", err)
	}
	s.Name = name

	var dhcpOptions []json.RawMessage
	if err := json.Unmarshal(tuple[1], &dhcpOptions); err != nil {
		return fmt.Errorf("failed to parse logic switch port DHCP options: %w", err)
	}
	if len(dhcpOptions) != 2 {
		return fmt.Errorf("invalid logic switch port DHCP options, expected 2 fields, got %d", len(dhcpOptions))
	}

	var optionType string
	if err := json.Unmarshal(dhcpOptions[0], &optionType); err != nil {
		return fmt.Errorf(
			"failed to parse logical switch port DHCP options type: %w",
			err,
		)
	}

	switch optionType {
	case "set":
		var values []json.RawMessage
		if err := json.Unmarshal(dhcpOptions[1], &values); err != nil {
			return fmt.Errorf("failed to parse logical switch port DHCP options set: %w", err)
		}
		if len(values) != 0 {
			return fmt.Errorf("invalid logical switch port DHCP options set, expected empty set, got %d values", len(values))
		}
		s.DHCPOptionsUUID = ""
	case "uuid":
		if err := json.Unmarshal(dhcpOptions[1], &s.DHCPOptionsUUID); err != nil {
			return fmt.Errorf(
				"failed to parse logical switch port DHCP options UUID: %w",
				err,
			)
		}
	default:
		return fmt.Errorf("invalid logic switch port DHCP options, expected 'set' or 'uuid', got %s", dhcpOptions[0])
	}

	return nil
}

func buildDHCPOptionEvidence(dhcpOptions []dhcp, ports []logicSwitchPort) []types.InspectionDHCPOptionEvidence {
	portsByDHCPUUID := make(map[string][]string)
	for _, port := range ports {
		if port.DHCPOptionsUUID != "" {
			portsByDHCPUUID[port.DHCPOptionsUUID] = append(portsByDHCPUUID[port.DHCPOptionsUUID], port.Name)
		}
	}

	evidence := make([]types.InspectionDHCPOptionEvidence, len(dhcpOptions))
	for i, option := range dhcpOptions {
		evidence[i] = types.InspectionDHCPOptionEvidence{
			UUID:                 option.UUID,
			CIDR:                 option.CIDR,
			ExternalIDs:          option.ExternalIDs,
			Ports:                portsByDHCPUUID[option.UUID],
			ClasslessStaticRoute: option.Options.ClasslessStaticRoutes,
		}
		slices.Sort(evidence[i].Ports)
	}
	slices.SortFunc(evidence, func(a, b types.InspectionDHCPOptionEvidence) int {
		return strings.Compare(a.UUID, b.UUID)
	})

	return evidence
}

func collectDHCP(ctx context.Context, s state.State) ([]types.InspectionDHCPOptionEvidence, []types.InspectionCollectionError) {
	var evidence []types.InspectionDHCPOptionEvidence
	var errors []types.InspectionCollectionError

	var dhcp dhcpOptionsResponse
	rawDHCP, err := probeDHCPOptions(ctx, s)
	if err != nil {
		errors = append(errors, types.InspectionCollectionError{
			FactGroup: "network",
			Message:   err.Error(),
		})
	} else {
		if err := json.Unmarshal([]byte(rawDHCP), &dhcp); err != nil {
			errors = append(errors, types.InspectionCollectionError{
				FactGroup: "network",
				Message:   fmt.Sprintf("failed to parse DHCP options: %v", err),
			})
		}
	}

	var lsp logicSwitchPortResponse
	rawLSP, err := probeLogicalSwitchPorts(ctx, s)
	if err != nil {
		errors = append(errors, types.InspectionCollectionError{
			FactGroup: "network",
			Message:   err.Error(),
		})
	} else {
		if err := json.Unmarshal([]byte(rawLSP), &lsp); err != nil {
			errors = append(errors, types.InspectionCollectionError{
				FactGroup: "network",
				Message:   fmt.Sprintf("failed to parse logical switch ports: %v", err),
			})
		}
	}

	evidence = buildDHCPOptionEvidence(dhcp.Data, lsp.Data)

	return evidence, errors
}

func probeDHCPOptions(ctx context.Context, s state.State) (string, error) {
	var args = []string{
		"--timeout", "1",
		"--format=json", "--columns=_uuid,cidr,options,external_ids",
		"list", "dhcp_options",
	}
	return cmd.NBCtlCluster(ctx, s, 1, args...)
}

func probeLogicalSwitchPorts(ctx context.Context, s state.State) (string, error) {
	var args = []string{
		"--timeout", "1",
		"--format=json", "--columns=name,dhcpv4_options",
		"list", "logical_switch_port",
	}

	return cmd.NBCtlCluster(ctx, s, 1, args...)
}
