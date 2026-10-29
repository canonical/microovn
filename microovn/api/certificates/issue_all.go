package certificates

import (
	"errors"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"net/http"

	"github.com/canonical/lxd/shared/logger"

	"github.com/canonical/microovn/microovn/securitylog"
)

// IssueCertificatesAllEndpoint defines endpoint for /1.0/certificates
var IssueCertificatesAllEndpoint = microTypes.Endpoint{
	Path: "certificates",
	Put:  microTypes.EndpointAction{Handler: issueCertificatesAllPut, AllowUntrusted: false, ProxyTarget: true},
}

// issueCertificatesAllPut implements PUT method for /1.0/certificates endpoint. The function issues new
// certificates for every OVN service enabled on this cluster member.
func issueCertificatesAllPut(s microTypes.State, r *http.Request) microTypes.Response {
	securitylog.Log(
		securitylog.CatAuthn,
		securitylog.EventPasswordChanged,
		logger.Ctx{"action": "issue_all_certificates"},
		"Bulk certificate reissue requested for all enabled OVN services",
	)
	logger.Info("Re-issuing certificate for all enabled OVN services.")
	responseData, err := reissueAllCertificates(r.Context(), s)
	if err != nil {
		logger.Errorf("Failed to issue certificates for all services: %v", err)
		return microTypes.InternalError(errors.New("internal server error"))
	}

	return microTypes.SyncResponse(true, responseData)
}
