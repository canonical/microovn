package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/rest"
	"github.com/canonical/microcluster/v2/state"
	"github.com/canonical/microovn/microovn/api/types"
	microOvnClient "github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/config"
)

// ConfigEndpoint - /1.0/config endpoint.
var ConfigEndoint = rest.Endpoint{
	Path:   "config",
	Get:    rest.EndpointAction{Handler: getConfig, AllowUntrusted: false, ProxyTarget: true},
	Post:   rest.EndpointAction{Handler: setConfig, AllowUntrusted: false, ProxyTarget: true},
	Delete: rest.EndpointAction{Handler: deleteConfig, AllowUntrusted: false, ProxyTarget: true},
}

// configHandler is a signature of a function that can be invoked on configuration option change.
type configHandler = func(ctx context.Context, s state.State, key string, value string) error

// spec is a structure that defines a valid configuration option
type spec struct {
	Key     string        // Name of the config option
	Handler configHandler // Optional function that will be executed on value change (may be nil)
}

// AllowedConfigKeys is a list of all valid configuration options
var AllowedConfigKeys = []spec{
	{Key: "ovn.central-ips", Handler: ovnCentralIpsUpdated},
}

// setConfig function handles configuration value changes submitted via POST request to config endpoint
func setConfig(s state.State, r *http.Request) response.Response {
	var configRequest types.SetConfigRequest
	configResponse := types.SetConfigResponse{}
	handler, err := parseConfigRequest(r, &configRequest)
	if err != nil {
		configResponse.Error = err.Error()
		return response.SyncResponse(false, &configResponse)
	}

	err = config.SetConfig(r.Context(), s, configRequest.Key, configRequest.Value)
	if err != nil {
		configResponse.Error = fmt.Sprintf("Error occurred while setting config: %v", err)
		return response.SyncResponse(false, &configResponse)
	}

	if handler != nil {
		err = handler(r.Context(), s, configRequest.Key, configRequest.Value)
		if err != nil {
			logger.Errorf(err.Error())
			configResponse.Error = fmt.Sprintf("Error occurred while handling config change: %v", err)
			return response.SyncResponse(false, &configResponse)
		}
	}

	return response.SyncResponse(true, &configResponse)
}

// getConfig handles GET requests to the config endpoint by returning the current config option value
func getConfig(s state.State, r *http.Request) response.Response {
	var configRequest types.GetConfigRequest
	configResponse := types.GetConfigResponse{}
	_, err := parseConfigRequest(r, &configRequest)
	if err != nil {
		configResponse.Error = err.Error()
		return response.SyncResponse(false, &configResponse)
	}

	item, err := config.GetConfig(r.Context(), s, configRequest.Key)
	if err != nil {
		configResponse.Error = fmt.Sprintf("Error occurred while getting config: %v", err)
		return response.SyncResponse(false, &configResponse)
	}

	if item == nil {
		configResponse.IsSet = false
	} else {
		configResponse.IsSet = true
		configResponse.Value = item.Value
	}

	return response.SyncResponse(true, &configResponse)
}

// deleteConfig handles DELETE requests to the config endpoint by completely removing the config option
func deleteConfig(s state.State, r *http.Request) response.Response {
	var configRequest types.DeleteConfigRequest
	configResponse := types.DeleteConfigResponse{}
	handler, err := parseConfigRequest(r, &configRequest)
	if err != nil {
		configResponse.Error = err.Error()
		return response.SyncResponse(false, &configResponse)
	}

	err = config.DeleteConfig(r.Context(), s, configRequest.Key)
	if err != nil {
		configResponse.Error = fmt.Sprintf("Error occurred while deleting config: %v", err)
		return response.SyncResponse(false, &configResponse)
	}

	if handler != nil {
		err = handler(r.Context(), s, configRequest.Key, "")
		if err != nil {
			logger.Errorf(err.Error())
			configResponse.Error = fmt.Sprintf("Error occurred while handling config change: %v", err)
		}
	}

	return response.SyncResponse(true, &configResponse)
}

// parseConfigRequest validates requests to the config endpoint. If the request is made for
// a valid config option, it returns the handler function associated with it.
// This function returns an error if it fails to parse the body of the request or if a request
// is made for an unknown configuration option.
func parseConfigRequest(r *http.Request, parsedData any) (configHandler, error) {
	err := json.NewDecoder(r.Body).Decode(&parsedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config request: %v", err)
	}
	var keyValue string
	switch v := parsedData.(type) {

	case *types.SetConfigRequest:
		keyValue = v.Key
	case *types.GetConfigRequest:
		keyValue = v.Key
	default:
		return nil, fmt.Errorf("unknown config request type")
	}

	allowedKey := false
	var handler configHandler
	for _, keySpec := range AllowedConfigKeys {
		if keySpec.Key == keyValue {
			allowedKey = true
			handler = keySpec.Handler
			break
		}
	}

	if !allowedKey {
		return nil, fmt.Errorf("config key '%s' is not a recognized config option", keyValue)
	}

	return handler, nil
}

// ovnCentralIpsUpdated is a handler for changes to the "ovn.central-ips" config option change. It triggers
// microovn.api.RegenerateEnvEndpoint to refresh controller configuration on every cluster member.
func ovnCentralIpsUpdated(ctx context.Context, s state.State, key string, _ string) error {
	errMsgPrefix := fmt.Sprintf("handling of '%s' config failed.", key)

	client, err := s.Leader()
	if err != nil {
		logger.Errorf("failed to trigger OVN environment refresh. %v", err)
		errMsg := fmt.Sprintf(
			"%s Failed to trigger environment update in the OVN cluster. "+
				"Config value was not successfully applied. Please see logs for more details.",
			errMsgPrefix,
		)
		return fmt.Errorf("%s", errMsg)
	}

	refreshResponse, err := microOvnClient.RegenerateEnvironment(ctx, client)
	if err != nil || !refreshResponse.Success {
		logger.Errorf("failed to refresh OVN environment. %v", err)
		logger.Errorf(strings.Join(refreshResponse.Errors, "\n"))
		errMsg := fmt.Sprintf(
			"%s Failed to trigger environment update in the OVN cluster. "+
				"Cluster may be in inconsistent state! Please see logs for more details.",
			errMsgPrefix,
		)
		return fmt.Errorf("%s", errMsg)
	}
	return err
}
