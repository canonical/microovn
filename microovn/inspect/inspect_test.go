package inspect

import (
	"context"
	"testing"
	"time"

	"github.com/canonical/microovn/microovn/api/types"
)

type fixedCheck struct {
	results []types.InspectionResult
}

func (fixedCheck) Name() string {
	return "fixed"
}

func (check fixedCheck) Run(context.Context, Input) []types.InspectionResult {
	return check.results
}

func boolPointer(value bool) *bool {
	return &value
}

func TestSummarize(t *testing.T) {
	results := []types.InspectionResult{
		{Status: types.InspectionStatusPass},
		{Status: types.InspectionStatusWarning},
		{Status: types.InspectionStatusUnknown},
		{Status: types.InspectionStatusFail},
	}

	summary := summarize(results)
	if summary.Status != types.InspectionStatusFail {
		t.Fatalf("summary status = %q, want %q", summary.Status, types.InspectionStatusFail)
	}
	if summary.Counts != (types.InspectionCounts{Pass: 1, Warning: 1, Fail: 1, Unknown: 1}) {
		t.Fatalf("summary counts = %#v", summary.Counts)
	}
}

func TestSummarizeSchema(t *testing.T) {
	tests := []struct {
		name       string
		evidence   types.InspectionSchemaEvidence
		wantStatus types.InspectionStatus
		responding int
	}{
		{
			name: "matching",
			evidence: types.InspectionSchemaEvidence{
				Database: "nb", ActiveVersion: "1.0",
				Members: []types.InspectionSchemaMemberEvidence{{Node: "node1", Version: "1.0"}},
			},
			wantStatus: types.InspectionStatusPass,
			responding: 1,
		},
		{
			name: "mismatch",
			evidence: types.InspectionSchemaEvidence{
				Database: "nb", ActiveVersion: "1.0",
				Members: []types.InspectionSchemaMemberEvidence{{Node: "node1", Version: "2.0"}},
			},
			wantStatus: types.InspectionStatusFail,
			responding: 1,
		},
		{
			name: "unsupported response",
			evidence: types.InspectionSchemaEvidence{
				Database: "nb", ActiveVersion: "1.0",
				Members: []types.InspectionSchemaMemberEvidence{{Node: "node1", Unsupported: true}},
			},
			wantStatus: types.InspectionStatusUnknown,
			responding: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			summary := summarizeSchema(test.evidence)
			if summary.Status != test.wantStatus {
				t.Fatalf("status = %q, want %q", summary.Status, test.wantStatus)
			}
			if summary.RespondingMembers != test.responding {
				t.Fatalf("responding members = %d, want %d", summary.RespondingMembers, test.responding)
			}
		})
	}
}

func TestSummarizeCommunication(t *testing.T) {
	tests := []struct {
		name       string
		evidence   types.InspectionCommunicationEvidence
		wantStatus types.InspectionStatus
	}{
		{
			name: "converged",
			evidence: types.InspectionCommunicationEvidence{
				NBReachable: boolPointer(true), SBReachable: boolPointer(true), Converged: boolPointer(true),
			},
			wantStatus: types.InspectionStatusPass,
		},
		{
			name: "not converged",
			evidence: types.InspectionCommunicationEvidence{
				NBReachable: boolPointer(true), SBReachable: boolPointer(true), Converged: boolPointer(false),
			},
			wantStatus: types.InspectionStatusWarning,
		},
		{
			name: "unreachable",
			evidence: types.InspectionCommunicationEvidence{
				NBReachable: boolPointer(false),
			},
			wantStatus: types.InspectionStatusFail,
		},
		{name: "unknown", wantStatus: types.InspectionStatusUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := summarizeCommunication(test.evidence).Status; got != test.wantStatus {
				t.Fatalf("status = %q, want %q", got, test.wantStatus)
			}
		})
	}
}

func TestBuildReportNonAuthoritative(t *testing.T) {
	timestamp := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.FixedZone("test", 3600))
	input := Input{ExecutionContext: types.InspectionExecutionContext{
		LocalNode: "node1", MemberRole: types.InspectionMemberRoleStandby, Scope: types.InspectionScopeLocal,
	}}
	check := fixedCheck{results: []types.InspectionResult{{
		ID: "check", Status: types.InspectionStatusUnknown,
		Details: []types.InspectionDetail{
			{Node: "node2", ID: "b"},
			{Node: "node1", ID: "b"},
			{Node: "node1", ID: "a"},
		},
	}}}

	report := buildReport(context.Background(), timestamp, input, []Check{check})
	if len(report.Results) != 2 || report.Results[0].ID != "execution-authority" || report.Results[1].ID != "check" {
		t.Fatalf("result order = %#v", report.Results)
	}
	details := report.Results[1].Details
	if details[0].Node != "node1" || details[0].ID != "a" || details[1].Node != "node1" || details[1].ID != "b" || details[2].Node != "node2" {
		t.Fatalf("detail order = %#v", details)
	}
	if report.Timestamp.Location() != time.UTC {
		t.Fatalf("timestamp location = %v, want UTC", report.Timestamp.Location())
	}
	if report.Summary.Status != types.InspectionStatusUnknown || report.Summary.Counts.Warning != 1 {
		t.Fatalf("summary = %#v", report.Summary)
	}
	if report.DatabaseSummary.Northbound.Status != types.InspectionStatusUnknown ||
		report.DatabaseSummary.Southbound.Status != types.InspectionStatusUnknown ||
		report.DatabaseSummary.Communication.Status != types.InspectionStatusUnknown {
		t.Fatalf("database summary = %#v", report.DatabaseSummary)
	}
	if report.DatabaseSummary.Northbound.Message != "cluster schema health is unavailable from a non-voting member" ||
		report.DatabaseSummary.Southbound.Message != "cluster schema health is unavailable from a non-voting member" ||
		report.DatabaseSummary.Communication.Message != "cluster database communication is unavailable from a non-voting member" {
		t.Fatalf("database summary messages = %#v", report.DatabaseSummary)
	}
}
