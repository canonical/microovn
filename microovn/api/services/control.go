package services

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/rest"
	"github.com/canonical/microcluster/v2/state"
	"github.com/gorilla/mux"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/node"
)

// ServiceControlCmd - /1.0/services/service endpoint.
var ServiceControlCmd = rest.Endpoint{
	Path:   "service/{service}",
	Delete: rest.EndpointAction{Handler: disableService, AllowUntrusted: false, ProxyTarget: true},
	Put:    rest.EndpointAction{Handler: enableService, AllowUntrusted: false, ProxyTarget: true},
}

// enableService - function to handle to service control put request,
// which aims to enable a service.
//
// This will return a response which contains a WarningSet for the
// current desired state and a response string on the operation
func enableService(s state.State, r *http.Request) response.Response {
	requestedService, err := url.PathUnescape(mux.Vars(r)["service"])
	if err != nil {
		logger.Errorf("Failed to get service: %s", err)
		return response.ErrorResponse(500, "Internal server error")
	}
	if !node.CheckValidService(requestedService) {
		return response.InternalError(errors.New("Service does not exist"))
	}
	err = node.EnableService(r.Context(), s, requestedService)
	if err != nil {
		return response.InternalError(err)
	}

	scr := types.ServiceControlResponse{}
	scr.Warnings, err = node.ServiceWarnings(r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to generate warnings for service: %s: %s", requestedService, err)
		return response.ErrorResponse(500, "Internal server error")
	}
	scr.Message = requestedService + " enabled"

	return response.SyncResponse(true, scr)
}

// disableService - function to handle to service control delete request,
// which aims to disable a service.
//
// This will return a response which contains a WarningSet for the
// current desired state and a response string on the operation
func disableService(s state.State, r *http.Request) response.Response {
	requestedService, err := url.PathUnescape(mux.Vars(r)["service"])
	if err != nil {
		logger.Errorf("Failed to get service: %s", err)
		return response.ErrorResponse(500, "Internal server error")
	}
	if !node.CheckValidService(requestedService) {
		return response.InternalError(errors.New("Service does not exist"))
	}
	err = node.DisableService(r.Context(), s, requestedService)
	if err != nil {
		return response.InternalError(err)
	}

	scr := types.ServiceControlResponse{}
	scr.Warnings, err = node.ServiceWarnings(r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to generate warnings for service: %s: %s", requestedService, err)
		return response.ErrorResponse(500, "Internal server error")
	}
	scr.Message = requestedService + " disabled"

	return response.SyncResponse(true, scr)
}
