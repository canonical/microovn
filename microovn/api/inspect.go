package api

import (
	"fmt"
	"net/http"

	"github.com/canonical/microcluster/v3/microcluster/rest"
	"github.com/canonical/microcluster/v3/microcluster/rest/response"
	"github.com/canonical/microcluster/v3/state"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/inspect/facts"
	"github.com/canonical/microovn/microovn/ovn/environment"
	"github.com/canonical/microovn/microovn/snap"
)

var InspectionSnapshotEndpoint = rest.Endpoint{
	Path: "inspect/snapshot",
	Get: rest.EndpointAction{
		Handler:        inspectionSnapshotGet,
		AllowUntrusted: false,
		ProxyTarget:    false,
	},
}

var InspectionDatabaseEndpoint = rest.Endpoint{
	Path: "inspect/database",
	Get: rest.EndpointAction{
		Handler:        inspectionDatabaseGet,
		AllowUntrusted: false,
		ProxyTarget:    false,
	},
}

var InspectionNetworkEndpoint = rest.Endpoint{
	Path: "inspect/network",
	Get: rest.EndpointAction{
		Handler:        inspectionNetworkGet,
		AllowUntrusted: false,
		ProxyTarget:    false,
	},
}

func inspectionSnapshotGet(s state.State, r *http.Request) response.Response {
	snapshot := facts.CollectNodeSnapshot(
		r.Context(),
		s.Name(),
		s.Address().Hostname(),
		snap.Services,
		environment.ReadEnvironment,
	)
	return response.SyncResponse(true, snapshot)
}

func inspectionDatabaseGet(s state.State, r *http.Request) response.Response {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		return response.BadRequest(fmt.Errorf("missing required query parameter: scope"))
	}
	if scope != string(types.InspectionScopeCluster) && scope != string(types.InspectionScopeLocal) {
		return response.BadRequest(fmt.Errorf("invalid scope: %s", scope))
	}
	probe, err := facts.CollectDatabaseProbe(
		r.Context(),
		s,
		types.InspectionScope(scope),
	)
	if err != nil {
		return response.BadRequest(err)
	}

	return response.SyncResponse(true, probe)
}

func inspectionNetworkGet(s state.State, r *http.Request) response.Response {
	probe := facts.CollectNetworkProbe(
		r.Context(),
		s,
	)
	return response.SyncResponse(true, probe)
}
