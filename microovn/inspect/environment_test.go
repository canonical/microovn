package inspect

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/canonical/microovn/microovn/api/types"
)

func TestEnvironmentCheck(t *testing.T) {
	environmentA := []types.InspectionEnvironment{{Name: "B", Hash: "hash-b"}, {Name: "A", Hash: "hash-a"}}
	environmentB := []types.InspectionEnvironment{{Name: "A", Hash: "hash-other"}, {Name: "C", Hash: "hash-c"}}

	tests := []struct {
		name         string
		input        Input
		wantIDs      []string
		wantStatuses []types.InspectionStatus
	}{
		{
			name: "non-voter includes local fingerprint",
			input: Input{
				ExecutionContext: types.InspectionExecutionContext{LocalNode: "node1"},
				Snapshots: map[string]types.InspectionNodeSnapshot{
					"node1": {NodeName: "node1", Environment: environmentA},
				},
			},
			wantIDs:      []string{"environment"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown},
		},
		{
			name:         "single member passes",
			input:        environmentInput([]string{"node1"}, map[string][]types.InspectionEnvironment{"node1": environmentA}),
			wantIDs:      []string{"environment"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass},
		},
		{
			name: "equal environments pass",
			input: environmentInput([]string{"node2", "node1"}, map[string][]types.InspectionEnvironment{
				"node1": environmentA,
				"node2": {{Name: "A", Hash: "hash-a"}, {Name: "B", Hash: "hash-b"}},
			}),
			wantIDs:      []string{"environment"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass},
		},
		{
			name: "member-specific environment differences pass",
			input: environmentInput([]string{"node1", "node2"}, map[string][]types.InspectionEnvironment{
				"node1": {
					{Name: "OVN_INITIAL_NB", Hash: "node2"},
					{Name: "OVN_INITIAL_SB", Hash: "node2"},
					{Name: "OVN_LOCAL_IP", Hash: "node1"},
					{Name: "OVN_NB_CONNECT", Hash: "shared"},
				},
				"node2": {
					{Name: "OVN_INITIAL_NB", Hash: "node1"},
					{Name: "OVN_INITIAL_SB", Hash: "node1"},
					{Name: "OVN_LOCAL_IP", Hash: "node2"},
					{Name: "OVN_NB_CONNECT", Hash: "shared"},
				},
			}),
			wantIDs:      []string{"environment"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass},
		},
		{
			name: "drift warns",
			input: environmentInput([]string{"node1", "node2"}, map[string][]types.InspectionEnvironment{
				"node1": environmentA,
				"node2": environmentB,
			}),
			wantIDs:      []string{"environment-drift"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusWarning},
		},
		{
			name:         "missing snapshot is unknown",
			input:        environmentInput([]string{"node1", "node2"}, map[string][]types.InspectionEnvironment{"node1": environmentA}),
			wantIDs:      []string{"environment-coverage"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown},
		},
		{
			name: "fact error is unknown",
			input: func() Input {
				input := environmentInput([]string{"node1"}, map[string][]types.InspectionEnvironment{"node1": environmentA})
				snapshot := input.Snapshots["node1"]
				snapshot.Errors = []types.InspectionCollectionError{{FactGroup: "environment", Message: "read failed"}}
				input.Snapshots["node1"] = snapshot
				return input
			}(),
			wantIDs:      []string{"environment-coverage"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown},
		},
		{
			name: "drift and missing coverage coexist",
			input: environmentInput([]string{"node3", "node1", "node2"}, map[string][]types.InspectionEnvironment{
				"node1": environmentA,
				"node2": environmentB,
			}),
			wantIDs:      []string{"environment-drift", "environment-coverage"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusWarning, types.InspectionStatusUnknown},
		},
		{
			name:         "missing member evidence is unknown",
			input:        environmentInput(nil, nil),
			wantIDs:      []string{"environment-coverage"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := (EnvironmentCheck{}).Run(context.Background(), test.input)
			if len(results) != len(test.wantIDs) {
				t.Fatalf("result count = %d, want %d: %#v", len(results), len(test.wantIDs), results)
			}
			for index, result := range results {
				if result.ID != test.wantIDs[index] || result.Status != test.wantStatuses[index] {
					t.Errorf("result %d = (%q, %q), want (%q, %q)", index, result.ID, result.Status, test.wantIDs[index], test.wantStatuses[index])
				}
				if result.Category != "environment" {
					t.Errorf("result %d category = %q, want environment", index, result.Category)
				}
			}
		})
	}
}

func TestEnvironmentCheckDeterministicGroupsAndSafeDiff(t *testing.T) {
	input := environmentInput([]string{"node3", "node1", "node2"}, map[string][]types.InspectionEnvironment{
		"node1": {{Name: "A", Hash: "hash-a"}, {Name: "B", Hash: "hash-b"}},
		"node2": {{Name: "B", Hash: "hash-b"}, {Name: "A", Hash: "hash-a"}},
		"node3": {{Name: "A", Hash: "hash-other"}, {Name: "C", Hash: "hash-c"}},
	})

	results := (EnvironmentCheck{}).Run(context.Background(), input)
	if len(results) != 1 || len(results[0].Details) != 2 {
		t.Fatalf("results = %#v", results)
	}
	if results[0].Details[0].Summary != "Baseline environment fingerprint shared by node1, node2" {
		t.Fatalf("baseline detail = %#v", results[0].Details[0])
	}
	data := results[0].Details[1].Data
	if data["changed.A.baseline"] != "hash-a" || data["changed.A.observed"] != "hash-other" ||
		data["removed.B"] != "hash-b" || data["added.C"] != "hash-c" {
		t.Fatalf("diff data = %#v", data)
	}
}

func TestEnvironmentCheckResultEvidence(t *testing.T) {
	t.Run("non-voter includes hash-only local evidence", func(t *testing.T) {
		input := Input{
			ExecutionContext: types.InspectionExecutionContext{LocalNode: "node1"},
			Snapshots: map[string]types.InspectionNodeSnapshot{
				"node1": {
					NodeName: "node1",
					Environment: []types.InspectionEnvironment{{
						Name: "OVN_NB_DB", Hash: "safe-hash",
					}},
				},
			},
		}

		result := (EnvironmentCheck{}).Run(context.Background(), input)[0]
		if len(result.Details) != 1 || result.Details[0].Data["OVN_NB_DB"] != "safe-hash" {
			t.Fatalf("details = %#v", result.Details)
		}
		data, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "raw-environment-value") {
			t.Fatalf("result exposed a raw environment value: %s", data)
		}
	})

	t.Run("missing snapshot preserves collection cause", func(t *testing.T) {
		input := environmentInput([]string{"node1", "node2"}, map[string][]types.InspectionEnvironment{
			"node1": {{Name: "A", Hash: "hash-a"}},
		})
		input.CollectionErrors = []types.InspectionCollectionError{
			{Node: "node2", FactGroup: "member", Message: "context canceled"},
		}

		result := (EnvironmentCheck{}).Run(context.Background(), input)[0]
		if result.CollectionError != "node2: context canceled" {
			t.Fatalf("collection error = %q", result.CollectionError)
		}
		if len(result.Details) != 1 || result.Details[0].Node != "node2" ||
			result.Details[0].Status != types.InspectionStatusUnknown {
			t.Fatalf("details = %#v", result.Details)
		}
	})

	t.Run("fact error is preserved", func(t *testing.T) {
		input := environmentInput([]string{"node1"}, map[string][]types.InspectionEnvironment{"node1": nil})
		snapshot := input.Snapshots["node1"]
		snapshot.Errors = []types.InspectionCollectionError{{FactGroup: "environment", Message: "read failed"}}
		input.Snapshots["node1"] = snapshot

		result := (EnvironmentCheck{}).Run(context.Background(), input)[0]
		if result.CollectionError != "node1: read failed" {
			t.Fatalf("collection error = %q", result.CollectionError)
		}
	})

	t.Run("pass has healthy summary and no remediation", func(t *testing.T) {
		result := (EnvironmentCheck{}).Run(context.Background(), environmentInput(
			[]string{"node1"},
			map[string][]types.InspectionEnvironment{"node1": {{Name: "A", Hash: "hash-a"}}},
		))[0]
		if result.Summary != "Member environments are consistent" || result.Remediation != "" {
			t.Fatalf("result = %#v", result)
		}
	})
}

func TestEnvironmentCheckNonVoterSnapshotError(t *testing.T) {
	input := Input{
		ExecutionContext: types.InspectionExecutionContext{LocalNode: "node1"},
		CollectionErrors: []types.InspectionCollectionError{{
			Node: "node1", FactGroup: "snapshot", Message: "snapshot unavailable",
		}},
	}

	result := (EnvironmentCheck{}).Run(context.Background(), input)[0]
	if result.CollectionError != "snapshot unavailable" {
		t.Fatalf("collection error = %q", result.CollectionError)
	}
}

func environmentInput(memberNames []string, environments map[string][]types.InspectionEnvironment) Input {
	members := make([]microTypes.ClusterMember, 0, len(memberNames))
	for _, memberName := range memberNames {
		members = append(members, microTypes.ClusterMember{
			ClusterMemberLocal: microTypes.ClusterMemberLocal{Name: memberName},
		})
	}

	snapshots := make(map[string]types.InspectionNodeSnapshot, len(environments))
	for node, environment := range environments {
		snapshots[node] = types.InspectionNodeSnapshot{NodeName: node, Environment: environment}
	}

	return Input{
		ExecutionContext: types.InspectionExecutionContext{LocalNode: "node1", Authoritative: true},
		Members:          members,
		Snapshots:        snapshots,
	}
}
