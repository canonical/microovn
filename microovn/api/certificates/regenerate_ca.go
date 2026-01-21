package certificates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v3/microcluster/rest"
	"github.com/canonical/microcluster/v3/microcluster/rest/response"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/canonical/microcluster/v3/state"

	"github.com/canonical/microovn/microovn/api/types"
	microovnClient "github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/ovn/certificates"
)

// RegenerateCaEndpoint defines endpoint for /1.0/ca
var RegenerateCaEndpoint = rest.Endpoint{
	Path: "ca",
	Put:  rest.EndpointAction{Handler: regenerateCaPut, AllowUntrusted: false, ProxyTarget: true},
	Post: rest.EndpointAction{Handler: setCaPost, AllowUntrusted: false, ProxyTarget: true},
	Get:  rest.EndpointAction{Handler: infoCaGet, AllowUntrusted: false, ProxyTarget: true},
}

// infoCaGet returns additional information about CA certificate
func infoCaGet(s state.State, r *http.Request) response.Response {
	autoRenew, err := certificates.IsCaRenewable(r.Context(), s)
	if err != nil {
		logger.Errorf("Error checking if CA is renewable: %v", err)
		errMsg := "Failed to get CA renewability. See logs for more details."
		return response.SyncResponse(false, types.CaInfo{AutoRenew: false, Error: errMsg})
	}
	return response.SyncResponse(true, types.CaInfo{AutoRenew: autoRenew})
}

// regenerateCaPut implements PUT method for /1.0/ca endpoint. The function issues new CA certificate
// and triggers re-issue of all service certificates on all MicroOVN cluster members
func regenerateCaPut(s state.State, r *http.Request) response.Response {
	responseData := types.NewRegenerateCaResponse()
	// Only one recipient of this request needs to update the CA in the shared DB
	if !microTypes.IsNotification(r) {
		logger.Info("Re-issuing CA certificate and private key")
		CaUpdate, err := certificates.GenerateNewCACertificate(r.Context(), s)
		if err != nil {
			logger.Errorf("Failed to generate new CA certificate: %v", err)
			responseData.NewCa = false
			return response.SyncResponse(false, &responseData)
		}

		responseData.NewCa = CaUpdate
	}

	return updateOvnClusterCertificates(s, r, responseData)
}

// setCaPost implements POST method for /1.0/ca endpoint. The function updates CA certificate
// from data provided by the user and triggers re-issue of all service certificates on all MicroOVN
// cluster members
func setCaPost(s state.State, r *http.Request) response.Response {
	responseData := types.NewRegenerateCaResponse()
	// Only one recipient of this request needs to update the CA in the shared DB
	if !microTypes.IsNotification(r) {
		var err error
		logger.Info("Updating CA certificate and private key from user provided data")
		var customCaRequest types.CustomCaRequest
		err = json.NewDecoder(r.Body).Decode(&customCaRequest)
		if err != nil {
			logger.Errorf("Failed to decode CA certificate and key from the request: %v", err)
			responseData.NewCa = false
			return response.SyncResponse(false, &responseData)
		}

		CaUpdate, err := certificates.SetNewCACertificate(r.Context(), s, customCaRequest.Certificate, customCaRequest.PrivateKey)
		if err != nil {
			logger.Errorf("Failed to set custom user-provided CA certificate: %v", err)
			responseData.NewCa = false
			return response.SyncResponse(false, &responseData)
		}

		responseData.NewCa = CaUpdate
	}

	return updateOvnClusterCertificates(s, r, responseData)
}

func updateOvnClusterCertificates(s state.State, r *http.Request, responseData *types.RegenerateCaResponse) response.Response {
	var err error
	// If this is the initial node that received the request, notify the rest of the nodes
	// in the cluster to update their OVN certificates
	if !microTypes.IsNotification(r) {
		if !responseData.NewCa {
			logger.Info("Cluster certificates do not need updating")
			return response.SyncResponse(true, &responseData)
		}
		// Get clients for rest of the cluster members
		cluster, err := s.Connect().Cluster(true)
		if err != nil {
			logger.Errorf("Failed to get a client for every cluster member: %v", err)
			return response.SyncResponse(false, &responseData)
		}

		// Bump rest of the cluster members to reissue their certificates with new CA
		err = cluster.Query(r.Context(), true, func(ctx context.Context, c microTypes.Client) error {
			clientURL := c.URL()
			logger.Infof("Requesting cluster member at '%s' to re-issue its OVN certificates", clientURL.String())
			result, err := microovnClient.RegenerateCA(ctx, c)
			if err != nil {
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
	err = certificates.DumpCA(r.Context(), s)
	if err != nil {
		logger.Errorf("%v", err)
		return response.SyncResponse(false, &responseData)
	}

	reissuedCertificates, err := reissueAllCertificates(r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to reissue certificates with new CA: %v", err)
	}
	responseData.ReissuedCertificates[s.Name()] = *reissuedCertificates

	return response.SyncResponse(true, &responseData)
}
