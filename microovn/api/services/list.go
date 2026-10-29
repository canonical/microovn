package services

import (
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"net/http"

	"github.com/canonical/microovn/microovn/node"
)

// ListCmd - /1.0/services endpoint.
var ListCmd = microTypes.Endpoint{
	Path: "services",

	Get: microTypes.EndpointAction{Handler: cmdServicesGet, ProxyTarget: true},
}

// cmdServicesGet - handles services endpoint functionality,
// by calling the respective function within node and then handling
// any errors it throws.
func cmdServicesGet(s microTypes.State, r *http.Request) microTypes.Response {
	services, err := node.ListServices(r.Context(), s)
	if err != nil {
		return microTypes.InternalError(err)
	}

	return microTypes.SyncResponse(true, services)
}
