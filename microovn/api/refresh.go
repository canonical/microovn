package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/canonical/lxd/shared/logger"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"

	"github.com/canonical/microovn/microovn/api/types"
	microovnClient "github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/ovn"
)

// RegenerateEnvEndpoint defines endpoint for /1.0/env
var RegenerateEnvEndpoint = microTypes.Endpoint{
	Path: "env",
	Post: microTypes.EndpointAction{Handler: regenerateEnvPost, AllowUntrusted: false, ProxyTarget: true},
}

// regenerateEnvPost implements POST method for /1.0/env endpoint.
// This function triggers and environment update on all MicroOVN cluster members.
// This is typically to be used with enabling and disabling entral services.
func regenerateEnvPost(s microTypes.State, r *http.Request) microTypes.Response {
	responseData := types.NewRegenerateEnvResponse()

	// Check that this is the initial node to recive this request
	if !microTypes.IsNotification(r) {
		logger.Infof("Understood notification, forwarding refresh env request")
		// Get clients for the rest of the cluster members
		cluster, err := s.Connect().Cluster(true)
		if err != nil {
			logger.Errorf("Failed to get a client for every cluster member: %v", err)
			responseData.Success = false
			return microTypes.SyncResponse(false, &responseData)
		}
		responseData.Success = true

		// Bump rest of the cluster members to regenerate their environment
		err = cluster.Query(r.Context(), true, func(ctx context.Context, c microTypes.Client) error {
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
			return microTypes.SmartError(err)
		}
	}

	logger.Info("Regenerating environment file")

	ovn.Refresh(r.Context(), s)
	return microTypes.SyncResponse(true, &responseData)
}
