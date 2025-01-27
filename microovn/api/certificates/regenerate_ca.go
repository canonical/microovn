package certificates

import (
	"context"
	"fmt"
	"net/http"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/client"
	"github.com/canonical/microcluster/rest"
	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/api/types"
	microovnClient "github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/ovn"
)

// RegenerateCaEndpoint defines endpoint for /1.0/ca
var RegenerateCaEndpoint = rest.Endpoint{
	Path: "ca",
	Put:  rest.EndpointAction{Handler: regenerateCaPut, AllowUntrusted: false, ProxyTarget: true},
}

// regenerateCaPut implements PUT method for /1.0/ca endpoint. The function issues new CA certificate
// and triggers re-issue of all service certificates on all MicroOVN cluster members
func regenerateCaPut(s *state.State, r *http.Request) response.Response {
	var err error
	responseData := types.NewRegenerateCaResponse()

	// Check that this is the initial node that received the request and recreate new CA certificate
	if !client.IsForwardedRequest(r) {
		// Only one recipient of this request needs to generate new CA
		logger.Info("Re-issuing CA certificate and private key")
		err = ovn.GenerateNewCACertificate(s)
		if err != nil {
			logger.Errorf("Failed to generate new CA certificate: %w", err)
			responseData.NewCa = false
			return response.SyncResponse(false, &responseData)
		} else {
			responseData.NewCa = true
		}

		// Get clients for rest of the cluster members
		cluster, err := s.Cluster(r)
		if err != nil {
			logger.Errorf("Failed to get a client for every cluster member: %w", err)
			return response.SyncResponse(false, &responseData)
		}

		// Bump rest of the cluster members to reissue their certificates with new CA
		err = cluster.Query(s.Context, true, func(ctx context.Context, c *client.Client) error {
			logger.Infof("Requesting cluster member at '%s' to re-issue its OVN certificates", c.URL())
			result, err := microovnClient.RegenerateCA(ctx, c)
			if err != nil {
				clientURL := c.URL()
				errMsg := fmt.Sprintf("failed to contact cluster member with address %q: %s", clientURL.String(), err)
				responseData.Errors = append(responseData.Errors, errMsg)
			} else {
				for host, service := range result.ReissuedCertificates {
					responseData.ReissuedCertificates[host] = service
				}
			}

			return nil
		})
		if err != nil {
			return response.SmartError(err)
		}
	}

	logger.Info("Re-issuing all local OVN certificates")
	err = ovn.DumpCA(s)
	if err != nil {
		logger.Errorf("%w", err)
		return response.SyncResponse(false, &responseData)
	}

	reissuedCertificates, err := reissueAllCertificates(s)
	if err != nil {
		logger.Errorf("Failed to reissue certificates with new CA: %w", err)
	}
	responseData.ReissuedCertificates[s.Name()] = *reissuedCertificates

	return response.SyncResponse(true, &responseData)
}
