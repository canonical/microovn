package inspect

import (
	"context"
	"reflect"
	"testing"

	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/canonical/microovn/microovn/api/types"
)

func TestServiceRuntimeCheck(t *testing.T) {
	tests := []struct {
		name         string
		input        Input
		wantIDs      []string
		wantStatuses []types.InspectionStatus
	}{
		{
			name: "healthy desired services pass",
			input: serviceRuntimeInput(
				[]string{"node1"},
				types.Services{
					{Service: types.SrvSwitch, Location: "node1"},
					{Service: types.SrvChassis, Location: "node1"},
					{Service: types.SrvCentral, Location: "node1"},
					{Service: types.SrvBgp, Location: "node1"},
				},
				map[string][]types.InspectionDaemonState{
					"node1": healthyMappedDaemons(),
				},
			),
			wantIDs:      []string{"service-runtime"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusPass},
		},
		{
			name: "inactive desired daemon fails",
			input: serviceRuntimeInput(
				[]string{"node1"},
				types.Services{{Service: types.SrvSwitch, Location: "node1"}},
				map[string][]types.InspectionDaemonState{
					"node1": {{Name: "microovn.switch", Enabled: true}},
				},
			),
			wantIDs:      []string{"service-runtime-failure"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusFail},
		},
		{
			name: "disabled desired daemon fails",
			input: serviceRuntimeInput(
				[]string{"node1"},
				types.Services{{Service: types.SrvSwitch, Location: "node1"}},
				map[string][]types.InspectionDaemonState{
					"node1": {{Name: "microovn.switch", Active: true}},
				},
			),
			wantIDs:      []string{"service-runtime-failure"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusFail},
		},
		{
			name: "missing desired daemon fails",
			input: serviceRuntimeInput(
				[]string{"node1"},
				types.Services{{Service: types.SrvSwitch, Location: "node1"}},
				map[string][]types.InspectionDaemonState{"node1": nil},
			),
			wantIDs:      []string{"service-runtime-failure"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusFail},
		},
		{
			name: "unexpected active daemon warns",
			input: serviceRuntimeInput(
				[]string{"node1"},
				nil,
				map[string][]types.InspectionDaemonState{
					"node1": {{Name: "microovn.switch", Active: true}},
				},
			),
			wantIDs:      []string{"service-runtime-drift"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusWarning},
		},
		{
			name: "unexpected enabled daemon warns",
			input: serviceRuntimeInput(
				[]string{"node1"},
				nil,
				map[string][]types.InspectionDaemonState{
					"node1": {{Name: "microovn.bird", Enabled: true}},
				},
			),
			wantIDs:      []string{"service-runtime-drift"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusWarning},
		},
		{
			name: "missing desired state is unknown",
			input: func() Input {
				input := serviceRuntimeInput(
					[]string{"node1"}, nil,
					map[string][]types.InspectionDaemonState{"node1": healthyMappedDaemons()},
				)
				input.DesiredStateAvailable = false
				input.CollectionErrors = []types.InspectionCollectionError{{FactGroup: "services", Message: "services unavailable"}}
				return input
			}(),
			wantIDs:      []string{"service-runtime-coverage"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown},
		},
		{
			name: "daemon collection error is unknown",
			input: func() Input {
				input := serviceRuntimeInput([]string{"node1"}, nil, map[string][]types.InspectionDaemonState{"node1": nil})
				snapshot := input.Snapshots["node1"]
				snapshot.Errors = []types.InspectionCollectionError{{FactGroup: "daemons", Message: "snap failed"}}
				input.Snapshots["node1"] = snapshot
				return input
			}(),
			wantIDs:      []string{"service-runtime-coverage"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown},
		},
		{
			name: "failure warning and unavailable peer coexist",
			input: func() Input {
				input := serviceRuntimeInput(
					[]string{"node3", "node1", "node2"},
					types.Services{{Service: types.SrvSwitch, Location: "node1"}},
					map[string][]types.InspectionDaemonState{
						"node1": {{Name: "microovn.switch", Enabled: true}},
						"node2": {{Name: "microovn.bird", Active: true, Enabled: true}},
					},
				)
				input.CollectionErrors = []types.InspectionCollectionError{{Node: "node3", FactGroup: "member", Message: "context deadline exceeded"}}
				return input
			}(),
			wantIDs:      []string{"service-runtime-failure", "service-runtime-drift", "service-runtime-coverage"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusFail, types.InspectionStatusWarning, types.InspectionStatusUnknown},
		},
		{
			name: "no member evidence is unknown",
			input: Input{
				ExecutionContext:      types.InspectionExecutionContext{Authoritative: true},
				DesiredStateAvailable: true,
			},
			wantIDs:      []string{"service-runtime-coverage"},
			wantStatuses: []types.InspectionStatus{types.InspectionStatusUnknown},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := (ServiceRuntimeCheck{}).Run(context.Background(), test.input)
			if len(results) != len(test.wantIDs) {
				t.Fatalf("result count = %d, want %d: %#v", len(results), len(test.wantIDs), results)
			}
			for index, result := range results {
				if result.ID != test.wantIDs[index] || result.Status != test.wantStatuses[index] {
					t.Errorf("result %d = (%q, %q), want (%q, %q)", index, result.ID, result.Status, test.wantIDs[index], test.wantStatuses[index])
				}
				if result.Category != "service-runtime" {
					t.Errorf("result %d category = %q, want service-runtime", index, result.Category)
				}
			}
		})
	}
}

func TestServiceRuntimeCheckNonVoter(t *testing.T) {
	input := Input{
		ExecutionContext: types.InspectionExecutionContext{LocalNode: "node1"},
		Snapshots: map[string]types.InspectionNodeSnapshot{
			"node1": {
				NodeName: "node1",
				Daemons:  []types.InspectionDaemonState{{Name: "microovn.switch", Active: true, Enabled: true}},
			},
		},
	}

	result := (ServiceRuntimeCheck{}).Run(context.Background(), input)[0]
	if result.Status != types.InspectionStatusUnknown || len(result.Details) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if got := result.Details[0].Data["microovn.switch"]; got != "active=true,enabled=true" {
		t.Fatalf("local daemon state = %q", got)
	}
}

func TestServiceRuntimeCheckResultContract(t *testing.T) {
	if name := (ServiceRuntimeCheck{}).Name(); name != "service-runtime" {
		t.Fatalf("name = %q, want service-runtime", name)
	}

	t.Run("missing snapshot preserves collection cause", func(t *testing.T) {
		input := serviceRuntimeInput([]string{"node1"}, nil, nil)
		input.CollectionErrors = []types.InspectionCollectionError{
			{Node: "node1", FactGroup: "member", Message: "context deadline exceeded"},
		}

		result := (ServiceRuntimeCheck{}).Run(context.Background(), input)[0]
		if result.CollectionError != "node1: context deadline exceeded" {
			t.Fatalf("collection error = %q", result.CollectionError)
		}
		if result.Remediation == "" {
			t.Fatal("unknown result has no remediation")
		}
	})

	t.Run("daemon fact error is preserved", func(t *testing.T) {
		input := serviceRuntimeInput([]string{"node1"}, nil, map[string][]types.InspectionDaemonState{"node1": nil})
		snapshot := input.Snapshots["node1"]
		snapshot.Errors = []types.InspectionCollectionError{{FactGroup: "daemons", Message: "snap failed"}}
		input.Snapshots["node1"] = snapshot

		result := (ServiceRuntimeCheck{}).Run(context.Background(), input)[0]
		if result.CollectionError != "node1: snap failed" {
			t.Fatalf("collection error = %q", result.CollectionError)
		}
	})

	t.Run("baseline daemon and stopped undesired daemon pass", func(t *testing.T) {
		input := serviceRuntimeInput(
			[]string{"node1"}, nil,
			map[string][]types.InspectionDaemonState{
				"node1": {
					{Name: "microovn.daemon", Active: true, Enabled: true},
					{Name: "microovn.switch"},
				},
			},
		)

		result := (ServiceRuntimeCheck{}).Run(context.Background(), input)[0]
		if result.Status != types.InspectionStatusPass || result.Remediation != "" {
			t.Fatalf("result = %#v", result)
		}
	})
}

func TestServiceRuntimeCheckDeterministicDetails(t *testing.T) {
	input := serviceRuntimeInput(
		[]string{"node2", "node1"},
		types.Services{
			{Service: types.SrvCentral, Location: "node2"},
			{Service: types.SrvCentral, Location: "node1"},
		},
		map[string][]types.InspectionDaemonState{
			"node1": {{Name: "microovn.ovn-ovsdb-server-sb"}},
			"node2": {{Name: "microovn.ovn-northd"}},
		},
	)

	result := (ServiceRuntimeCheck{}).Run(context.Background(), input)[0]
	var got []string
	for _, detail := range result.Details {
		got = append(got, detail.Node+":"+detail.ID)
	}
	want := []string{
		"node1:microovn.ovn-northd",
		"node1:microovn.ovn-ovsdb-server-nb",
		"node1:microovn.ovn-ovsdb-server-sb",
		"node2:microovn.ovn-northd",
		"node2:microovn.ovn-ovsdb-server-nb",
		"node2:microovn.ovn-ovsdb-server-sb",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("detail order = %#v, want %#v", got, want)
	}
}

func serviceRuntimeInput(memberNames []string, services types.Services, daemons map[string][]types.InspectionDaemonState) Input {
	members := make([]microTypes.ClusterMember, 0, len(memberNames))
	for _, memberName := range memberNames {
		members = append(members, microTypes.ClusterMember{
			ClusterMemberLocal: microTypes.ClusterMemberLocal{Name: memberName},
		})
	}

	snapshots := make(map[string]types.InspectionNodeSnapshot, len(daemons))
	for node, states := range daemons {
		snapshots[node] = types.InspectionNodeSnapshot{NodeName: node, Daemons: states}
	}

	return Input{
		ExecutionContext:      types.InspectionExecutionContext{Authoritative: true},
		Services:              services,
		Members:               members,
		Snapshots:             snapshots,
		DesiredStateAvailable: true,
	}
}

func healthyMappedDaemons() []types.InspectionDaemonState {
	return []types.InspectionDaemonState{
		{Name: "microovn.switch", Active: true, Enabled: true},
		{Name: "microovn.chassis", Active: true, Enabled: true},
		{Name: "microovn.ovn-northd", Active: true, Enabled: true},
		{Name: "microovn.ovn-ovsdb-server-nb", Active: true, Enabled: true},
		{Name: "microovn.ovn-ovsdb-server-sb", Active: true, Enabled: true},
		{Name: "microovn.bird", Active: true, Enabled: true},
	}
}
