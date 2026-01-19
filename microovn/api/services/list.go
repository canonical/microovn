package services

import (
	"net/http"

	"github.com/canonical/microcluster/v3/microcluster/rest"
	"github.com/canonical/microcluster/v3/microcluster/rest/response"
	"github.com/canonical/microcluster/v3/state"

	"github.com/canonical/microovn/microovn/node"
)

// ListCmd - /1.0/services endpoint.
var ListCmd = rest.Endpoint{
	Path: "services",

	Get: rest.EndpointAction{Handler: cmdServicesGet, ProxyTarget: true},
}

// cmdServicesGet - handles services endpoint functionality,
// by calling the respective function within node and then handling
// any errors it throws.
func cmdServicesGet(s state.State, r *http.Request) response.Response {
	services, err := node.ListServices(r.Context(), s)
	if err != nil {
		return response.InternalError(err)
	}

	return response.SyncResponse(true, services)
}
