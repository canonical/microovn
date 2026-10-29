package services

import (
	"encoding/json"
	"errors"
	"fmt"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"net/http"
	"net/url"

	"github.com/canonical/lxd/shared/logger"
	"github.com/gorilla/mux"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/node"
)

// ServiceControlCmd - /1.0/services/service endpoint.
var ServiceControlCmd = microTypes.Endpoint{
	Path:   "service/{service}",
	Delete: microTypes.EndpointAction{Handler: disableService, AllowUntrusted: false, ProxyTarget: true},
	Put:    microTypes.EndpointAction{Handler: enableService, AllowUntrusted: false, ProxyTarget: true},
}

// enableService - function to handle to service control put request,
// which aims to enable a service.
//
// This will return a response which contains a WarningSet for the
// current desired state and a response string on the operation
func enableService(s microTypes.State, r *http.Request) microTypes.Response {
	requestedService, err := url.PathUnescape(mux.Vars(r)["service"])
	if err != nil {
		logger.Errorf("Failed to get service: %s", err)
		return microTypes.InternalError(errors.New("internal server error"))
	}
	if !types.CheckValidService(requestedService) {
		return microTypes.InternalError(errors.New("service does not exist"))
	}

	var extraConfig types.ExtraServiceConfig
	err = json.NewDecoder(r.Body).Decode(&extraConfig)
	if err != nil {
		logger.Errorf("Failed to decode request body: %s", err)
		return microTypes.BadRequest(errors.New("failed to decode request"))
	}
	err = node.EnableService(r.Context(), s, requestedService, &extraConfig)
	if err != nil {
		return microTypes.InternalError(err)
	}

	scr := types.ServiceControlResponse{}
	scr.Warnings, err = node.ServiceWarnings(r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to generate warnings for service: %s: %s", requestedService, err)
		return microTypes.InternalError(errors.New("internal server error"))
	}
	scr.Message = requestedService + " enabled"

	return microTypes.SyncResponse(true, scr)
}

// disableService - function to handle to service control delete request,
// which aims to disable a service.
//
// This will return a response which contains a WarningSet for the
// current desired state and a response string on the operation
func disableService(s microTypes.State, r *http.Request) microTypes.Response {
	requestedService, err := url.PathUnescape(mux.Vars(r)["service"])
	if err != nil {
		logger.Errorf("Failed to get service: %s", err)
		return microTypes.InternalError(errors.New("internal server error"))
	}

	if !types.CheckValidService(requestedService) {
		return microTypes.InternalError(errors.New("service does not exist"))
	}

	var requestData types.DisableServiceRequest
	err = json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		return microTypes.InternalError(fmt.Errorf("failed to decode request: %w", err))
	}

	err = node.DisableService(r.Context(), s, requestedService, requestData.AllowDisableLastCentral)
	if err != nil {
		return microTypes.InternalError(err)
	}

	scr := types.ServiceControlResponse{}
	scr.Warnings, err = node.ServiceWarnings(r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to generate warnings for service: %s: %s", requestedService, err)
		return microTypes.InternalError(errors.New("internal server error"))
	}
	scr.Message = requestedService + " disabled"

	return microTypes.SyncResponse(true, scr)
}
