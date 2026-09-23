package inspect

import (
	"context"
	"testing"

	"github.com/canonical/microovn/microovn/api/types"
)

func TestTopologyCheck(t *testing.T) {
	tests := []struct {
		name                string
		input               Input
		wantIDs             []string
		wantStatuses        []types.InspectionStatus
		wantCollectionError string
	}{
		{
			name: "non-authoritative",
			input: Input{ExecutionContext: types.InspectionExecutionContext{
				MemberRole: types.InspectionMemberRoleStandby,
			}},
			wantIDs:      []string{"central-topology"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown},
		},
		{
			name: "desired state unavailable prefers services error",
			input: Input{
				ExecutionContext:      types.InspectionExecutionContext{Authoritative: true},
				DesiredStateAvailable: false,
				CollectionErrors: []types.InspectionCollectionError{
					{FactGroup: "member", Message: "member error"},
					{FactGroup: "services", Message: "services error"},
				},
			},
			wantIDs:             []string{"central-topology"},
			wantStatuses:        []types.InspectionStatus{types.InspectionStatusUnknown},
			wantCollectionError: "services error",
		},
		{
			name:         "zero central is exempt",
			input:        topologyInput(0),
			wantIDs:      []string{"central-topology"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass},
		},
		{
			name:         "one central is too few",
			input:        topologyInput(1),
			wantIDs:      []string{"central-count-few"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusWarning},
		},
		{
			name:         "two centrals are even and too few",
			input:        topologyInput(2),
			wantIDs:      []string{"central-count-even", "central-count-few"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusWarning, types.InspectionStatusWarning},
		},
		{
			name:         "three centrals are healthy",
			input:        topologyInput(3),
			wantIDs:      []string{"central-topology"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass},
		},
		{
			name:         "four centrals are even",
			input:        topologyInput(4),
			wantIDs:      []string{"central-count-even"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusWarning},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := (TopologyCheck{}).Run(context.Background(), test.input)
			if len(results) != len(test.wantIDs) {
				t.Fatalf("result count = %d, want %d: %#v", len(results), len(test.wantIDs), results)
			}

			for index, result := range results {
				if result.ID != test.wantIDs[index] || result.Status != test.wantStatuses[index] {
					t.Errorf("result %d = (%q, %q), want (%q, %q)", index, result.ID, result.Status, test.wantIDs[index], test.wantStatuses[index])
				}
			}
			if results[0].CollectionError != test.wantCollectionError {
				t.Errorf("collection error = %q, want %q", results[0].CollectionError, test.wantCollectionError)
			}
		})
	}
}

func TestTopologyCheckResultContract(t *testing.T) {
	if name := (TopologyCheck{}).Name(); name != "topology" {
		t.Fatalf("name = %q, want topology", name)
	}

	results := (TopologyCheck{}).Run(context.Background(), topologyInput(2))
	for _, result := range results {
		if result.Category != "cluster" {
			t.Errorf("category = %q, want cluster", result.Category)
		}
		if result.Remediation == "" {
			t.Errorf("result %q has no remediation", result.ID)
		}
	}

	input := Input{
		ExecutionContext: types.InspectionExecutionContext{Authoritative: true},
		CollectionErrors: []types.InspectionCollectionError{{FactGroup: "member", Message: "member error"}},
	}
	results = (TopologyCheck{}).Run(context.Background(), input)
	if results[0].CollectionError != "member error" {
		t.Fatalf("collection error = %q, want member error", results[0].CollectionError)
	}
}

func topologyInput(centralCount int) Input {
	services := make(types.Services, centralCount)
	for index := range services {
		services[index] = types.Service{Service: types.SrvCentral}
	}

	return Input{
		ExecutionContext:      types.InspectionExecutionContext{Authoritative: true},
		Services:              services,
		DesiredStateAvailable: true,
	}
}
