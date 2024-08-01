package certificates

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/v2/rest"
	"github.com/canonical/microcluster/v2/state"
	"github.com/gorilla/mux"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/ovn"
)

// IssueCertificatesEndpoint defines endpoint for /1.0/certificates/<service-name>.
var IssueCertificatesEndpoint = rest.Endpoint{
	Path: "certificates/{service}",
	Put:  rest.EndpointAction{Handler: issueCertificatesPut, AllowUntrusted: false, ProxyTarget: true},
}

// issueCertificatesPut implements PUT method for /1.0/certificates/<service-name> endpoint. The function parses
// service name from the request URL and if the service is currently enabled on this cluster member, it
// issues new certificate for it.
func issueCertificatesPut(s *state.State, r *http.Request) response.Response {
	// Get requested service name
	requestedService, err := url.PathUnescape(mux.Vars(r)["service"])
	if err != nil {
		logger.Errorf("failed to parse service name from URL '%s'", r.URL)
		return response.ErrorResponse(500, "Internal server error")
	}
	logger.Infof("Issuing new certificate for '%s' service.", requestedService)

	// Get all enabled services and make sure that the requested service is among them.
	eligibleServices, err := enabledOvnServices(s)
	if err != nil {
		logger.Errorf("failed to lookup local services eligible for certificate refresh: %s", err)
		return response.ErrorResponse(500, "Internal server error.")
	}

	isCertificateAllowed := false
	for _, service := range eligibleServices {
		if requestedService == service {
			isCertificateAllowed = true
			break
		}
	}

	// Fail with 404 if requested service is not enabled
	if !isCertificateAllowed {
		missingMsg := fmt.Sprintf(
			"Can't issue certificate for service '%s'. Service is not enabled on this member. Enabled services: %s",
			requestedService,
			strings.Join(eligibleServices, ", "),
		)
		logger.Warn(missingMsg)
		return response.ErrorResponse(404, missingMsg)
	}

	// Attempt to issue new certificate and return response object
	err = ovn.GenerateNewServiceCertificate(s, requestedService, ovn.CertificateTypeServer)
	result := types.IssueCertificateResponse{}

	if err != nil {
		result.Failed = []string{requestedService}
		logger.Errorf("failed to reissue certificate for '%s' service: %s", requestedService, err)
		return response.SyncResponse(false, &result)
	}

	result.Success = []string{requestedService}

	return response.SyncResponse(true, &result)
}
