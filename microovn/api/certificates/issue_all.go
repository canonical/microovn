package certificates

import (
	"errors"
	"net/http"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v3/microcluster/rest"
	"github.com/canonical/microcluster/v3/microcluster/rest/response"
	"github.com/canonical/microcluster/v3/state"
)

// IssueCertificatesAllEndpoint defines endpoint for /1.0/certificates
var IssueCertificatesAllEndpoint = rest.Endpoint{
	Path: "certificates",
	Put:  rest.EndpointAction{Handler: issueCertificatesAllPut, AllowUntrusted: false, ProxyTarget: true},
}

// issueCertificatesAllPut implements PUT method for /1.0/certificates endpoint. The function issues new
// certificates for every OVN service enabled on this cluster member.
func issueCertificatesAllPut(s state.State, r *http.Request) response.Response {
	logger.Info("Re-issuing certificate for all enabled OVN services.")
	responseData, err := reissueAllCertificates(r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to issue certificates for all services: %v", err)
		return response.InternalError(errors.New("internal server error"))
	}

	return response.SyncResponse(true, responseData)
}
