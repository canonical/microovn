package inspect

import (
	"context"
	"testing"

	"github.com/canonical/microovn/microovn/api/types"
)

func TestNetworkCheck(t *testing.T) {
	tests := []struct {
		name                string
		input               Input
		wantID              string
		wantStatus          types.InspectionStatus
		wantSummary         string
		wantDetails         int
		wantCollectionError string
	}{
		{
			name:       "non-authoritative",
			input:      Input{},
			wantID:     "network",
			wantStatus: types.InspectionStatusUnknown,
		},
		{
			name: "collection errors",
			input: networkInput(nil, []types.InspectionCollectionError{
				{Message: "DHCP options unavailable"},
				{Message: "logical switch ports unavailable"},
			}),
			wantID:              "network",
			wantStatus:          types.InspectionStatusUnknown,
			wantCollectionError: "DHCP options unavailable; logical switch ports unavailable",
		},
		{
			name: "network endpoint error",
			input: Input{
				ExecutionContext: types.InspectionExecutionContext{Authoritative: true},
				CollectionErrors: []types.InspectionCollectionError{
					{FactGroup: "database", Message: "database unavailable"},
					{FactGroup: "network", Message: "failed to get network probe"},
				},
			},
			wantID:              "network",
			wantStatus:          types.InspectionStatusUnknown,
			wantCollectionError: "failed to get network probe",
		},
		{
			name:        "no applicable options",
			input:       networkInput(nil, nil),
			wantID:      "network",
			wantStatus:  types.InspectionStatusPass,
			wantSummary: "No applicable DHCPv4 options were found",
		},
		{
			name: "valid metadata route",
			input: networkInput([]types.InspectionDHCPOptionEvidence{{
				UUID:                 "dhcp-a",
				CIDR:                 "10.0.0.0/24",
				Ports:                []string{"port-a"},
				ClasslessStaticRoute: []string{"0.0.0.0/0,10.0.0.1", "169.254.169.254/32,10.0.0.2"},
			}}, nil),
			wantID:      "network",
			wantStatus:  types.InspectionStatusPass,
			wantSummary: "DHCP metadata routes are configured",
		},
		{
			name: "unreferenced and IPv6 options are ignored",
			input: networkInput([]types.InspectionDHCPOptionEvidence{
				{UUID: "dhcp-a", CIDR: "10.0.0.0/24"},
				{UUID: "dhcp-b", CIDR: "fd42::/64", Ports: []string{"port-b"}},
			}, nil),
			wantID:      "network",
			wantStatus:  types.InspectionStatusPass,
			wantSummary: "No applicable DHCPv4 options were found",
		},
		{
			name: "missing metadata route",
			input: networkInput([]types.InspectionDHCPOptionEvidence{{
				UUID:                 "dhcp-a",
				CIDR:                 "10.0.0.0/24",
				Ports:                []string{"port-a", "port-b"},
				ClasslessStaticRoute: []string{"0.0.0.0/0,10.0.0.1"},
			}}, nil),
			wantID:      "dhcp-metadata-route",
			wantStatus:  types.InspectionStatusFail,
			wantDetails: 1,
		},
		{
			name: "invalid metadata next hop",
			input: networkInput([]types.InspectionDHCPOptionEvidence{{
				UUID:                 "dhcp-a",
				CIDR:                 "10.0.0.0/24",
				Ports:                []string{"port-a"},
				ClasslessStaticRoute: []string{"169.254.169.254/32,not-an-address"},
			}}, nil),
			wantID:      "dhcp-metadata-route",
			wantStatus:  types.InspectionStatusFail,
			wantDetails: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := (NetworkCheck{}).Run(context.Background(), test.input)
			if len(results) != 1 {
				t.Fatalf("result count = %d, want 1: %#v", len(results), results)
			}

			result := results[0]
			if result.ID != test.wantID || result.Status != test.wantStatus {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.ID, result.Status, test.wantID, test.wantStatus)
			}
			if test.wantSummary != "" && result.Summary != test.wantSummary {
				t.Errorf("summary = %q, want %q", result.Summary, test.wantSummary)
			}
			if len(result.Details) != test.wantDetails {
				t.Errorf("detail count = %d, want %d", len(result.Details), test.wantDetails)
			}
			if result.CollectionError != test.wantCollectionError {
				t.Errorf("collection error = %q, want %q", result.CollectionError, test.wantCollectionError)
			}
		})
	}
}

func TestNetworkCheckResultContract(t *testing.T) {
	if name := (NetworkCheck{}).Name(); name != "network" {
		t.Fatalf("name = %q, want network", name)
	}

	input := networkInput([]types.InspectionDHCPOptionEvidence{{
		UUID:  "dhcp-a",
		CIDR:  "10.0.0.0/24",
		Ports: []string{"port-a", "port-b"},
	}}, nil)
	result := (NetworkCheck{}).Run(context.Background(), input)[0]
	if result.Category != "network" {
		t.Errorf("category = %q, want network", result.Category)
	}
	if result.Remediation == "" {
		t.Error("failure has no remediation")
	}
	if len(result.Details) != 1 {
		t.Fatalf("detail count = %d, want 1", len(result.Details))
	}
	if result.Details[0].Data["uuid"] != "dhcp-a" || result.Details[0].Data["ports"] != "port-a,port-b" {
		t.Errorf("detail data = %#v", result.Details[0].Data)
	}
}

func networkInput(options []types.InspectionDHCPOptionEvidence, collectionErrors []types.InspectionCollectionError) Input {
	return Input{
		ExecutionContext: types.InspectionExecutionContext{Authoritative: true},
		Network: types.InspectionNetworkProbe{
			DHCPOptions:      options,
			CollectionErrors: collectionErrors,
		},
	}
}
