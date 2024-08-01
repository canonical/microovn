package services

import (
	"net/http"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/microcluster/v2/rest"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/node"
)

// ListCmd - /1.0/services endpoint.
var ListCmd = rest.Endpoint{
	Path: "services",

	Get: rest.EndpointAction{Handler: cmdServicesGet, ProxyTarget: true},
}

func cmdServicesGet(s *state.State, _ *http.Request) response.Response {
	services, err := node.ListServices(s)
	if err != nil {
		return response.InternalError(err)
	}

	return response.SyncResponse(true, services)
}
