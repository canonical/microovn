package inspect

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/canonical/microcluster/v3/microcluster"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/canonical/microovn/microovn/api/types"
)

type inspectionClient struct {
	query func(context.Context, any) error
}

func (client inspectionClient) URL() *url.URL {
	return nil
}

func (client inspectionClient) HTTP() *http.Client {
	return nil
}

func (client inspectionClient) Query(ctx context.Context, _ string, _ microTypes.EndpointPrefix, _ *url.URL, _ any, out any) error {
	return client.query(ctx, out)
}

func (inspectionClient) QueryRaw(context.Context, string, microTypes.EndpointPrefix, *url.URL, any) (*http.Response, error) {
	return nil, errors.New("not implemented")
}

func (inspectionClient) Websocket(context.Context, microTypes.EndpointPrefix, *url.URL) (*websocket.Conn, error) {
	return nil, errors.New("not implemented")
}

func (inspectionClient) SetClusterNotification() {}

func (client inspectionClient) UseTarget(string) microTypes.Client {
	return client
}

func TestResolveExecutionContext(t *testing.T) {
	statusAddress, err := microTypes.ParseAddrPort("10.0.0.1:8443")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		status        *microTypes.Server
		members       []microTypes.DqliteMember
		wantNode      string
		wantRole      types.InspectionMemberRole
		authoritative bool
		wantError     bool
	}{
		{
			name:          "matches voter by name",
			status:        &microTypes.Server{Name: "node1", Address: statusAddress},
			members:       []microTypes.DqliteMember{{Name: "node1", Address: "10.0.0.1:9000", Role: "voter"}},
			wantNode:      "node1",
			wantRole:      "voter",
			authoritative: true,
		},
		{
			name:     "falls back to normalized host address",
			status:   &microTypes.Server{Name: "renamed", Address: statusAddress},
			members:  []microTypes.DqliteMember{{Name: "node1", Address: "10.0.0.1:9000", Role: "stand-by"}},
			wantNode: "node1",
			wantRole: "standby",
		},
		{
			name:      "rejects ambiguous name",
			status:    &microTypes.Server{Name: "node1", Address: statusAddress},
			members:   []microTypes.DqliteMember{{Name: "node1", Role: "voter"}, {Name: "node1", Role: "stand-by"}},
			wantError: true,
		},
		{
			name:      "rejects unsupported role",
			status:    &microTypes.Server{Name: "node1", Address: statusAddress},
			members:   []microTypes.DqliteMember{{Name: "node1", Role: "pending"}},
			wantError: true,
		},
		{
			name:      "rejects missing member",
			status:    &microTypes.Server{Name: "node1", Address: statusAddress},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			execution, err := resolveExecutionContext(test.status, test.members)
			if test.wantError {
				if err == nil {
					t.Fatalf("resolveExecutionContext returned %#v, want error", execution)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if execution.LocalNode != test.wantNode || execution.MemberRole != test.wantRole || execution.Authoritative != test.authoritative {
				t.Fatalf("execution context = %#v", execution)
			}
		})
	}
}

func TestDqliteMembersError(t *testing.T) {
	t.Run("missing cluster state is uninitialized", func(t *testing.T) {
		err := dqliteMembersError(&os.PathError{Op: "open", Path: "cluster.yaml", Err: os.ErrNotExist})
		if err.Error() != "MicroOVN is not initialized" {
			t.Fatalf("error = %q", err)
		}
	})

	t.Run("other errors preserve cause", func(t *testing.T) {
		cause := errors.New("read failed")
		err := dqliteMembersError(cause)
		if !errors.Is(err, cause) {
			t.Fatalf("error %q does not preserve cause", err)
		}
	})
}

func TestGetMemberNodeSnapshotsPreservesPartialResults(t *testing.T) {
	members := []memberClient{
		{
			member: microTypes.ClusterMember{ClusterMemberLocal: microTypes.ClusterMemberLocal{Name: "node2"}},
			client: inspectionClient{query: func(context.Context, any) error { return errors.New("peer unavailable") }},
		},
		{
			member: microTypes.ClusterMember{ClusterMemberLocal: microTypes.ClusterMemberLocal{Name: "node1"}},
			client: inspectionClient{query: func(_ context.Context, out any) error {
				*out.(*types.InspectionNodeSnapshot) = types.InspectionNodeSnapshot{NodeName: "wrong-name"}
				return nil
			}},
		},
	}

	snapshots, collectionErrors := getMemberNodeSnapshots(context.Background(), members)
	if len(snapshots) != 1 || snapshots[0].NodeName != "node1" {
		t.Fatalf("snapshots = %#v", snapshots)
	}
	if len(collectionErrors) != 1 || collectionErrors[0].Node != "node2" {
		t.Fatalf("collection errors = %#v", collectionErrors)
	}
}

func TestGetMemberNodeSnapshotsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var queryCalled atomic.Bool
	members := []memberClient{{
		member: microTypes.ClusterMember{ClusterMemberLocal: microTypes.ClusterMemberLocal{Name: "node1"}},
		client: inspectionClient{query: func(context.Context, any) error {
			queryCalled.Store(true)
			return nil
		}},
	}}

	snapshots, collectionErrors := getMemberNodeSnapshots(ctx, members)
	if len(snapshots) != 0 || len(collectionErrors) != 1 {
		t.Fatalf("snapshots = %#v, errors = %#v", snapshots, collectionErrors)
	}
	if collectionErrors[0].Message == "" {
		t.Fatal("cancellation error is empty")
	}
	if queryCalled.Load() {
		t.Fatal("query called after cancellation")
	}
}

func TestOrchestrateUsesInjectedCollectionAndChecks(t *testing.T) {
	input := Input{
		ExecutionContext: types.InspectionExecutionContext{Authoritative: true},
		Schemas:          make(map[string]types.InspectionSchemaEvidence),
	}
	checks := []Check{
		fixedCheck{results: []types.InspectionResult{{ID: "first", Status: types.InspectionStatusPass}}},
		fixedCheck{results: []types.InspectionResult{{ID: "second", Status: types.InspectionStatusWarning}}},
	}
	collectCalls := 0

	report, err := Orchestrate(context.Background(), Dependencies{
		Checks: checks,
		CollectInput: func(context.Context, *microcluster.MicroCluster, microTypes.Client) (Input, error) {
			collectCalls++
			return input, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if collectCalls != 1 {
		t.Fatalf("collect calls = %d, want 1", collectCalls)
	}
	var resultIDs []string
	for _, result := range report.Results {
		resultIDs = append(resultIDs, result.ID)
	}
	if !reflect.DeepEqual(resultIDs, []string{"first", "second"}) {
		t.Fatalf("result order = %#v", resultIDs)
	}
	if report.Summary.Status != types.InspectionStatusWarning {
		t.Fatalf("summary = %#v", report.Summary)
	}
}
