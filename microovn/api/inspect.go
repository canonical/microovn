package api

import (
	"fmt"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"net/http"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/inspect/facts"
	"github.com/canonical/microovn/microovn/ovn/environment"
	"github.com/canonical/microovn/microovn/snap"
)

var InspectionSnapshotEndpoint = microTypes.Endpoint{
	Path: "inspect/snapshot",
	Get: microTypes.EndpointAction{
		Handler:        inspectionSnapshotGet,
		AllowUntrusted: false,
		ProxyTarget:    false,
	},
}

var InspectionDatabaseEndpoint = microTypes.Endpoint{
	Path: "inspect/database",
	Get: microTypes.EndpointAction{
		Handler:        inspectionDatabaseGet,
		AllowUntrusted: false,
		ProxyTarget:    false,
	},
}

var InspectionNetworkEndpoint = microTypes.Endpoint{
	Path: "inspect/network",
	Get: microTypes.EndpointAction{
		Handler:        inspectionNetworkGet,
		AllowUntrusted: false,
		ProxyTarget:    false,
	},
}

func inspectionSnapshotGet(s microTypes.State, r *http.Request) microTypes.Response {
	snapshot := facts.CollectNodeSnapshot(
		r.Context(),
		s.Name(),
		s.Address().Hostname(),
		snap.Services,
		environment.ReadEnvironment,
	)
	return microTypes.SyncResponse(true, snapshot)
}

func inspectionDatabaseGet(s microTypes.State, r *http.Request) microTypes.Response {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		return microTypes.BadRequest(fmt.Errorf("missing required query parameter: scope"))
	}
	if scope != string(types.InspectionScopeCluster) && scope != string(types.InspectionScopeLocal) {
		return microTypes.BadRequest(fmt.Errorf("invalid scope: %s", scope))
	}
	probe, err := facts.CollectDatabaseProbe(
		r.Context(),
		s,
		types.InspectionScope(scope),
	)
	if err != nil {
		return microTypes.BadRequest(err)
	}

	return microTypes.SyncResponse(true, probe)
}

func inspectionNetworkGet(s microTypes.State, r *http.Request) microTypes.Response {
	probe := facts.CollectNetworkProbe(
		r.Context(),
		s,
	)
	return microTypes.SyncResponse(true, probe)
}
