package services

import (
	"net/http"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/microcluster/rest"
	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/node"
)

// /1.0/services endpoint.
var ListCmd = rest.Endpoint{
	Path: "services",

	Get: rest.EndpointAction{Handler: cmdServicesGet, ProxyTarget: true},
}

func cmdServicesGet(s *state.State, r *http.Request) response.Response {
	services, err := node.ListServices(s)
	if err != nil {
		return response.InternalError(err)
	}

	return response.SyncResponse(true, services)
}
