package inspect

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/canonical/microovn/microovn/api/types"
)

func testReport() types.InspectionReport {
	nbCfg := int64(8)
	sbCfg := int64(7)

	return types.InspectionReport{
		SchemaVersion: 1,
		ExecutionContext: types.InspectionExecutionContext{
			LocalNode:     "node1",
			MemberRole:    "voter",
			Authoritative: true,
			Scope:         "cluster",
		},
		Summary: types.InspectionSummary{
			Status: types.InspectionStatusWarning,
			Counts: types.InspectionCounts{Pass: 1, Warning: 1},
		},
		DatabaseSummary: types.InspectionDatabaseSummary{
			Northbound: types.InspectionDatabaseStatus{
				Status: types.InspectionStatusPass, ActiveSchemaVersion: "7.3.0",
				ExpectedMembers: 3, RespondingMembers: 3,
			},
			Southbound: types.InspectionDatabaseStatus{
				Status: types.InspectionStatusWarning, ActiveSchemaVersion: "20.30.0",
				ExpectedMembers: 3, RespondingMembers: 2, Message: "one peer unavailable",
			},
			Communication: types.InspectionCommunicationSummary{
				Status: types.InspectionStatusWarning, NBCfg: &nbCfg, SBCfg: &sbCfg,
				Message: "not converged",
			},
		},
		Results: []types.InspectionResult{
			{ID: "topology", Category: "cluster", Status: types.InspectionStatusPass, Summary: "Topology is healthy"},
			{
				ID: "environment", Category: "environment", Status: types.InspectionStatusWarning,
				Summary: "Environment differs", Remediation: "Compare generated configuration",
				Details: []types.InspectionDetail{{
					Node: "node2", ID: "changed", Status: types.InspectionStatusWarning,
					Summary: "Key differs", Data: map[string]string{"hash": "def", "key": "OVN_KEY"},
				}},
			},
		},
	}
}

func TestRenderTextConcise(t *testing.T) {
	var output bytes.Buffer
	report := testReport()

	err := RenderText(&output, report, false)
	if err != nil {
		t.Fatalf("RenderText returned an error: %v", err)
	}

	want := "Execution: node=node1 role=voter scope=cluster authoritative=true\n" +
		"Database:\n" +
		"  Northbound: PASS active_schema=7.3.0 responding=3/3\n" +
		"  Southbound: WARNING active_schema=20.30.0 responding=2/3 message=\"one peer unavailable\"\n" +
		"  Communication: WARNING nb_cfg=8 sb_cfg=7 message=\"not converged\"\n" +
		"[WARNING] environment/environment: Environment differs\n" +
		"Summary: WARNING (pass=1 warning=1 fail=0 unknown=0)\n"
	if output.String() != want {
		t.Fatalf("unexpected concise output:\n%s\nwant:\n%s", output.String(), want)
	}
}

func TestRenderTextVerboseDoesNotMutateReport(t *testing.T) {
	var output bytes.Buffer
	report := testReport()
	before := testReport()

	err := RenderText(&output, report, true)
	if err != nil {
		t.Fatalf("RenderText returned an error: %v", err)
	}
	if !reflect.DeepEqual(report, before) {
		t.Fatal("RenderText mutated the report")
	}

	wantLines := []string{
		"[PASS] cluster/topology: Topology is healthy",
		"  [WARNING] node2/changed: Key differs",
		"    hash=def",
		"    key=OVN_KEY",
		"  Remediation: Compare generated configuration",
	}
	for _, line := range wantLines {
		if !strings.Contains(output.String(), line+"\n") {
			t.Fatalf("verbose output does not contain %q:\n%s", line, output.String())
		}
	}
}

func TestRenderTextVerboseDetailWithoutNode(t *testing.T) {
	var output bytes.Buffer
	report := testReport()
	report.Results[1].Details = []types.InspectionDetail{{
		ID: "database-communication", Status: types.InspectionStatusPass,
		Summary: "Database communication is healthy",
	}}

	err := RenderText(&output, report, true)
	if err != nil {
		t.Fatalf("RenderText returned an error: %v", err)
	}
	if !strings.Contains(output.String(), "  [PASS] database-communication: Database communication is healthy\n") {
		t.Fatalf("verbose output has an invalid detail label:\n%s", output.String())
	}
}

func TestRenderJSON(t *testing.T) {
	var output bytes.Buffer
	report := testReport()

	err := RenderJSON(&output, report)
	if err != nil {
		t.Fatalf("RenderJSON returned an error: %v", err)
	}
	if strings.Count(output.String(), "\n") != 1 || !strings.HasSuffix(output.String(), "\n") {
		t.Fatalf("JSON output must have exactly one trailing newline: %q", output.String())
	}

	var decoded types.InspectionReport
	err = json.Unmarshal(output.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}
	if !reflect.DeepEqual(decoded, report) {
		t.Fatalf("decoded report differs from input: %#v", decoded)
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		status types.InspectionStatus
		want   int
	}{
		{status: types.InspectionStatusPass, want: 0},
		{status: types.InspectionStatusWarning, want: 1},
		{status: types.InspectionStatusFail, want: 1},
		{status: types.InspectionStatusUnknown, want: 1},
	}

	for _, test := range tests {
		t.Run(string(test.status), func(t *testing.T) {
			report := types.InspectionReport{Summary: types.InspectionSummary{Status: test.status}}
			if got := ExitCode(report); got != test.want {
				t.Fatalf("ExitCode returned %d, want %d", got, test.want)
			}
		})
	}
}
