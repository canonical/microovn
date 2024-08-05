package services

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/microcluster/rest"
	"github.com/canonical/microcluster/state"
	"github.com/gorilla/mux"

	"github.com/canonical/microovn/microovn/node"
)

// /1.0/services/service endpoint.
var ServiceControlCmd = rest.Endpoint{
	Path:   "service/{service}",
	Delete: rest.EndpointAction{Handler: disableService, AllowUntrusted: false, ProxyTarget: true},
	Put:    rest.EndpointAction{Handler: enableService, AllowUntrusted: false, ProxyTarget: true},
}

func enableService(s *state.State, r *http.Request) response.Response {
	requestedService, err := url.PathUnescape(mux.Vars(r)["service"])
	if err != nil {
		return response.InternalError(err)
	}
	if !node.CheckValidService(requestedService) {
		return response.InternalError(errors.New("Service does not exist"))
	}
	err = node.EnableService(s, requestedService)
	if err != nil {
		return response.InternalError(err)
	}
	return response.SyncResponse(true, requestedService+" enabled")
}

func disableService(s *state.State, r *http.Request) response.Response {
	requestedService, err := url.PathUnescape(mux.Vars(r)["service"])
	if err != nil {
		return response.InternalError(err)
	}
	if !node.CheckValidService(requestedService) {
		return response.InternalError(errors.New("Service does not exist"))
	}
	err = node.DisableService(s, requestedService)
	if err != nil {
		return response.InternalError(err)
	}
	return response.SyncResponse(true, requestedService+" disabled")
}
