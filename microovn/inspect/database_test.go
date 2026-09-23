package inspect

import (
	"context"
	"reflect"
	"testing"

	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/canonical/microovn/microovn/api/types"
)

func TestDatabaseCheck(t *testing.T) {
	tests := []struct {
		name         string
		input        Input
		wantIDs      []string
		wantStatuses []types.InspectionStatus
	}{
		{
			name:         "healthy",
			input:        healthyDatabaseInput(),
			wantIDs:      []string{"database-schema-nb", "database-schema-sb", "database-communication"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass, types.InspectionStatusPass, types.InspectionStatusPass},
		},
		{
			name: "mismatch and incomplete coverage coexist",
			input: func() Input {
				input := healthyDatabaseInput()
				input.Schemas["nb"] = types.InspectionSchemaEvidence{
					Database: "nb", ActiveVersion: "2.0",
					Members: []types.InspectionSchemaMemberEvidence{
						{Node: "node1", Version: "1.0"},
						{Node: "node2", Error: "member unreachable"},
					},
				}
				return input
			}(),
			wantIDs: []string{
				"database-schema-mismatch-nb",
				"database-schema-coverage-nb",
				"database-schema-sb",
				"database-communication",
			},
			wantStatuses: []types.InspectionStatus{
				types.InspectionStatusFail,
				types.InspectionStatusUnknown,
				types.InspectionStatusPass,
				types.InspectionStatusPass,
			},
		},
		{
			name: "unsupported API is unknown",
			input: func() Input {
				input := healthyDatabaseInput()
				input.Schemas["nb"].Members[0] = types.InspectionSchemaMemberEvidence{Node: "node1", Unsupported: true}
				return input
			}(),
			wantIDs:      []string{"database-schema-coverage-nb", "database-schema-sb", "database-communication"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown, types.InspectionStatusPass, types.InspectionStatusPass},
		},
		{
			name: "missing schema evidence is unknown",
			input: func() Input {
				input := healthyDatabaseInput()
				delete(input.Schemas, "nb")
				return input
			}(),
			wantIDs:      []string{"database-schema-coverage-nb", "database-schema-sb", "database-communication"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown, types.InspectionStatusPass, types.InspectionStatusPass},
		},
		{
			name: "communication lag warns",
			input: func() Input {
				input := healthyDatabaseInput()
				input.Communication.Converged = boolPointer(false)
				return input
			}(),
			wantIDs:      []string{"database-schema-nb", "database-schema-sb", "database-communication"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass, types.InspectionStatusPass, types.InspectionStatusWarning},
		},
		{
			name: "unreachable database fails",
			input: func() Input {
				input := healthyDatabaseInput()
				input.Communication.NBReachable = boolPointer(false)
				input.Communication.SBReachable = nil
				input.Communication.Converged = nil
				return input
			}(),
			wantIDs:      []string{"database-schema-nb", "database-schema-sb", "database-communication"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass, types.InspectionStatusPass, types.InspectionStatusFail},
		},
		{
			name: "incomplete communication is unknown",
			input: func() Input {
				input := healthyDatabaseInput()
				input.Communication.Converged = nil
				input.Communication.CollectionError = "probe canceled"
				return input
			}(),
			wantIDs:      []string{"database-schema-nb", "database-schema-sb", "database-communication"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass, types.InspectionStatusPass, types.InspectionStatusUnknown},
		},
		{
			name: "missing member list is unknown",
			input: func() Input {
				input := healthyDatabaseInput()
				input.Members = nil
				return input
			}(),
			wantIDs:      []string{"database-schema-coverage-nb", "database-schema-coverage-sb", "database-communication"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown, types.InspectionStatusUnknown, types.InspectionStatusPass},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := (DatabaseCheck{}).Run(context.Background(), test.input)
			if len(results) != len(test.wantIDs) {
				t.Fatalf("result count = %d, want %d: %#v", len(results), len(test.wantIDs), results)
			}
			for index, result := range results {
				if result.ID != test.wantIDs[index] || result.Status != test.wantStatuses[index] {
					t.Errorf("result %d = (%q, %q), want (%q, %q)", index, result.ID, result.Status, test.wantIDs[index], test.wantStatuses[index])
				}
				if result.Category != "database" {
					t.Errorf("result %d category = %q, want database", index, result.Category)
				}
			}
		})
	}
}

func TestDatabaseCheckNonVoter(t *testing.T) {
	input := Input{
		ExecutionContext: types.InspectionExecutionContext{LocalNode: "node1"},
		Schemas: map[string]types.InspectionSchemaEvidence{
			"nb": {Database: "nb", ActiveVersion: "1.0"},
		},
		Communication: types.InspectionCommunicationEvidence{
			NBReachable: boolPointer(true),
			NBCfg:       int64Pointer(42),
		},
	}

	result := (DatabaseCheck{}).Run(context.Background(), input)[0]
	if result.Status != types.InspectionStatusUnknown || len(result.Details) != 2 {
		t.Fatalf("result = %#v", result)
	}
	if result.Details[0].Data["active_version"] != "1.0" {
		t.Fatalf("schema detail = %#v", result.Details[0])
	}
	if result.Details[1].Data["nb_cfg"] != "42" {
		t.Fatalf("communication detail = %#v", result.Details[1])
	}
}

func TestDatabaseCheckDeterministicDetails(t *testing.T) {
	input := healthyDatabaseInput()
	input.Members[0], input.Members[1] = input.Members[1], input.Members[0]
	input.Schemas["nb"] = types.InspectionSchemaEvidence{
		Database: "nb", ActiveVersion: "2.0",
		Members: []types.InspectionSchemaMemberEvidence{
			{Node: "node2", Version: "1.0"},
			{Node: "node1", Version: "1.0"},
		},
	}

	result := (DatabaseCheck{}).Run(context.Background(), input)[0]
	var got []string
	for _, detail := range result.Details {
		got = append(got, detail.Node+":"+detail.ID)
	}
	want := []string{"node1:schema-version", "node2:schema-version"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("detail order = %#v, want %#v", got, want)
	}
}

func TestDatabaseCheckResultContract(t *testing.T) {
	if name := (DatabaseCheck{}).Name(); name != "database" {
		t.Fatalf("name = %q, want database", name)
	}

	input := healthyDatabaseInput()
	input.Schemas["nb"].Members[0] = types.InspectionSchemaMemberEvidence{Node: "node1", Error: "member unreachable"}
	result := (DatabaseCheck{}).Run(context.Background(), input)[0]
	if result.CollectionError != "node1: member unreachable" {
		t.Fatalf("collection error = %q", result.CollectionError)
	}
	if result.Remediation == "" {
		t.Fatal("unknown result has no remediation")
	}

	input = healthyDatabaseInput()
	input.Communication.Converged = nil
	input.Communication.CollectionError = "probe canceled"
	results := (DatabaseCheck{}).Run(context.Background(), input)
	if results[2].CollectionError != "probe canceled" || results[2].Remediation == "" {
		t.Fatalf("communication result = %#v", results[2])
	}

	input = healthyDatabaseInput()
	delete(input.Schemas, "nb")
	input.CollectionErrors = []types.InspectionCollectionError{{FactGroup: "database", Message: "schema request failed"}}
	result = (DatabaseCheck{}).Run(context.Background(), input)[0]
	if result.CollectionError != "schema request failed" {
		t.Fatalf("top-level collection error = %q", result.CollectionError)
	}
}

func healthyDatabaseInput() Input {
	members := []microTypes.ClusterMember{
		{ClusterMemberLocal: microTypes.ClusterMemberLocal{Name: "node1"}},
		{ClusterMemberLocal: microTypes.ClusterMemberLocal{Name: "node2"}},
	}
	memberEvidence := []types.InspectionSchemaMemberEvidence{
		{Node: "node1", Version: "1.0"},
		{Node: "node2", Version: "1.0"},
	}

	return Input{
		ExecutionContext: types.InspectionExecutionContext{Authoritative: true},
		Members:          members,
		Schemas: map[string]types.InspectionSchemaEvidence{
			"nb": {Database: "nb", ActiveVersion: "1.0", Members: append([]types.InspectionSchemaMemberEvidence(nil), memberEvidence...)},
			"sb": {Database: "sb", ActiveVersion: "1.0", Members: append([]types.InspectionSchemaMemberEvidence(nil), memberEvidence...)},
		},
		Communication: types.InspectionCommunicationEvidence{
			NBCfg:       int64Pointer(12),
			SBCfg:       int64Pointer(12),
			NBReachable: boolPointer(true),
			SBReachable: boolPointer(true),
			Converged:   boolPointer(true),
		},
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
