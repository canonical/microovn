package certificates

import (
	"errors"
	"fmt"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/node"
	"github.com/canonical/microovn/microovn/ovn"
)

// enabledOvnServices returns list of OVN services enabled on this MicroOVN cluster member.
func enabledOvnServices(s *state.State) ([]string, error) {
	var enabledServices []string
	var wrappedError error

	hasCentral, err := node.HasServiceActive(s, "central")
	if err != nil {
		wrappedError = errors.Join(wrappedError, fmt.Errorf("failed to lookup local services eligible for certificate refresh: %s", err))
	}

	hasSwitch, err := node.HasServiceActive(s, "switch")
	if err != nil {
		wrappedError = errors.Join(wrappedError, fmt.Errorf("failed to lookup local services eligible for certificate refresh: %s", err))
	}

	if hasCentral {
		enabledServices = append(enabledServices, "ovnnb", "ovnsb", "ovn-northd")
	}

	if hasSwitch {
		enabledServices = append(enabledServices, "ovn-controller")
	}

	// We always want a client certificate
	enabledServices = append(enabledServices, "client")

	return enabledServices, wrappedError
}

// reissueAllCertificates issues new certificates, using current CA, for every OVN service that is enabled
// on this MicroOVN cluster member.
func reissueAllCertificates(s *state.State) (*types.IssueCertificateResponse, error) {
	responseData := types.IssueCertificateResponse{}

	activeServices, err := enabledOvnServices(s)
	if err != nil {
		return nil, err
	}

	for _, service := range activeServices {
		err = ovn.GenerateNewServiceCertificate(s, service, ovn.CertificateTypeServer)
		if err != nil {
			logger.Errorf("Failed to issue certificate for %s: %s", service, err)
			responseData.Failed = append(responseData.Failed, service)
		} else {
			responseData.Success = append(responseData.Success, service)
		}
	}

	return &responseData, nil
}
