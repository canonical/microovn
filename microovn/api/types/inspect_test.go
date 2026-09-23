package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestInspectionJSONContract(t *testing.T) {
	timestamp := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	report := InspectionReport{
		SchemaVersion: InspectionSchemaVersion,
		Timestamp:     timestamp,
		ExecutionContext: InspectionExecutionContext{
			LocalNode:     "node1",
			MemberRole:    InspectionMemberRoleVoter,
			Authoritative: true,
			Scope:         InspectionScopeCluster,
		},
		Summary: InspectionSummary{
			Status: InspectionStatusWarning,
			Counts: InspectionCounts{Pass: 1, Warning: 1},
		},
		DatabaseSummary: InspectionDatabaseSummary{
			Northbound: InspectionDatabaseStatus{Status: InspectionStatusPass},
			Southbound: InspectionDatabaseStatus{Status: InspectionStatusWarning},
			Communication: InspectionCommunicationSummary{
				Status: InspectionStatusPass,
			},
		},
		Results: []InspectionResult{
			{ID: "first", Category: "cluster", Status: InspectionStatusPass, Summary: "first result"},
			{ID: "second", Category: "database", Status: InspectionStatusWarning, Summary: "second result"},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal inspection report: %v", err)
	}

	var wire map[string]json.RawMessage
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatalf("failed to decode inspection report: %v", err)
	}
	for _, key := range []string{"schema_version", "timestamp", "execution_context", "summary", "database_summary", "results"} {
		if _, ok := wire[key]; !ok {
			t.Errorf("inspection report is missing JSON field %q", key)
		}
	}
	if len(wire) != 6 {
		t.Errorf("inspection report has %d top-level fields, want 6", len(wire))
	}

	var encodedTimestamp string
	if err := json.Unmarshal(wire["timestamp"], &encodedTimestamp); err != nil {
		t.Fatalf("failed to decode timestamp: %v", err)
	}
	if encodedTimestamp != "2026-01-02T03:04:05Z" {
		t.Errorf("timestamp is %q, want UTC RFC3339 encoding", encodedTimestamp)
	}

	var results []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(wire["results"], &results); err != nil {
		t.Fatalf("failed to decode results: %v", err)
	}
	if len(results) != 2 || results[0].ID != "first" || results[1].ID != "second" {
		t.Errorf("result order changed: %#v", results)
	}

	rawValue := "ssl:private-fixture-value"
	hash := sha256.Sum256([]byte(rawValue))
	snapshot := InspectionNodeSnapshot{
		NodeName: "node1",
		Environment: []InspectionEnvironment{{
			Name: "OVN_KEY",
			Hash: fmt.Sprintf("%x", hash),
		}},
		Errors: []InspectionCollectionError{{
			FactGroup: "environment",
			Message:   "unavailable",
		}},
	}

	snapshotData, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("failed to marshal inspection snapshot: %v", err)
	}
	if bytes.Contains(snapshotData, []byte(rawValue)) {
		t.Fatal("serialized snapshot contains a raw environment value")
	}
	if bytes.Contains(snapshotData, []byte(`"value"`)) {
		t.Fatal("serialized snapshot exposes an environment value field")
	}
	if bytes.Contains(snapshotData, []byte(`"node"`)) {
		t.Fatal("snapshot-local collection error includes an empty node field")
	}
}
