package facts

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestBuildDHCPOptionEvidence(t *testing.T) {
	dhcpJSON := `{"data":[[["uuid","uuid-b"],"10.0.1.0/24",["map",[]],["map",[]]],[["uuid","uuid-a"],"10.0.0.0/24",["map",[["classless_static_route","{169.254.169.254/32,10.0.0.2, 0.0.0.0/0,10.0.0.1}"]]],["map",[["subnet_id","subnet-a"]]]]]}`
	portsJSON := `{"data":[["port-b",["uuid","uuid-a"]],["unused",["set",[]]],["port-a",["uuid","uuid-a"]]]}`

	var dhcpOptions dhcpOptionsResponse
	if err := json.Unmarshal([]byte(dhcpJSON), &dhcpOptions); err != nil {
		t.Fatalf("failed to decode DHCP options: %v", err)
	}
	var ports logicSwitchPortResponse
	if err := json.Unmarshal([]byte(portsJSON), &ports); err != nil {
		t.Fatalf("failed to decode logical switch ports: %v", err)
	}

	evidence := buildDHCPOptionEvidence(dhcpOptions.Data, ports.Data)
	if len(evidence) != 2 || evidence[0].UUID != "uuid-a" || evidence[1].UUID != "uuid-b" {
		t.Fatalf("evidence is not sorted by UUID: %+v", evidence)
	}
	if !slices.Equal(evidence[0].Ports, []string{"port-a", "port-b"}) {
		t.Fatalf("ports = %v, want [port-a port-b]", evidence[0].Ports)
	}
	wantRoutes := []string{"169.254.169.254/32,10.0.0.2", "0.0.0.0/0,10.0.0.1"}
	if !slices.Equal(evidence[0].ClasslessStaticRoute, wantRoutes) {
		t.Fatalf("classless static routes = %v, want %v", evidence[0].ClasslessStaticRoute, wantRoutes)
	}
}

func TestDHCPRejectsMalformedTaggedValues(t *testing.T) {
	tests := map[string]string{
		"UUID": `[["set",[]],"10.0.0.0/24",["map",[]],["map",[]]]`,
		"map":  `[["uuid","uuid-a"],"10.0.0.0/24",["set",[]],["map",[]]]`,
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			var option dhcp
			if err := json.Unmarshal([]byte(input), &option); err == nil {
				t.Fatalf("accepted malformed %s tagged value", name)
			}
		})
	}
}

func TestLogicalSwitchPortRejectsNonEmptyOptionalSet(t *testing.T) {
	var port logicSwitchPort
	err := json.Unmarshal([]byte(`["port-a",["set",[["uuid","uuid-a"]]]]`), &port)
	if err == nil {
		t.Fatal("accepted a non-empty DHCP options set")
	}
	if !strings.Contains(err.Error(), "expected empty set") {
		t.Fatalf("unexpected error: %v", err)
	}
}
