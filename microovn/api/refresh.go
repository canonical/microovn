package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/client"
	"github.com/canonical/microcluster/v2/rest"
	"github.com/canonical/microcluster/v2/state"

	"github.com/canonical/microovn/microovn/api/types"
	microovnClient "github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/ovn"
)

// RegenerateEnvEndpoint defines endpoint for /1.0/env
var RegenerateEnvEndpoint = rest.Endpoint{
	Path: "env",
	Post: rest.EndpointAction{Handler: regenerateEnvPost, AllowUntrusted: false, ProxyTarget: true},
}

// regenerateEnvPost implements POST method for /1.0/env endpoint.
// This function triggers and environment update on all MicroOVN cluster members.
// This is typically to be used with enabling and disabling entral services.
func regenerateEnvPost(s state.State, r *http.Request) response.Response {
	var err error
	responseData := types.NewRegenerateEnvResponse()

	// Check that this is the initial node to recive this request
	if !client.IsNotification(r) {
		logger.Infof("Understood notification, forwarding refresh env request")
		// Get clients for the rest of the cluster members
		cluster, err := s.Cluster(true)
		if err != nil {
			logger.Errorf("Failed to get a client for every cluster member: %v", err)
			responseData.Success = false
			return response.SyncResponse(false, &responseData)
		}
		responseData.Success = true

		// Bump rest of the cluster members to regenerate their environment
		err = cluster.Query(r.Context(), true, func(ctx context.Context, c *client.Client) error {
			clientURL := c.URL()
			logger.Infof("Requesting cluster member at '%s' to regenerate its environment file", clientURL.String())

			_, err := microovnClient.RegenerateEnvironment(ctx, c)

			if err != nil {
				errMsg := fmt.Sprintf("Failed to contact cluster member with address %q: %s", clientURL.String(), err)
				responseData.Errors = append(responseData.Errors, errMsg)
			}
			return nil
		})
		if err != nil {
			return response.SmartError(err)
		}
	}

	logger.Info("Regenerating environment file")

	err = ovn.Refresh(context.Background(), r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to refresh environment: %s", err)
		return response.SyncResponse(false, &responseData)
	}
	return response.SyncResponse(true, &responseData)
}
