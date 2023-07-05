package certificates

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/state"

	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/database"
	"github.com/canonical/microovn/microovn/ovn"
)

// enabledOvnServices returns list of OVN services enabled on this MicroOVN cluster member.
func enabledOvnServices(s *state.State) ([]string, error) {
	var enabledServices []string

	// Get list of existing local OVN services.
	err := s.Database.Transaction(s.Context, func(ctx context.Context, tx *sql.Tx) error {
		name := s.Name()
		services, err := database.GetServices(ctx, tx, database.ServiceFilter{Member: &name})
		if err != nil {
			return err
		}

		for _, srv := range services {
			if srv.Service == "central" {
				enabledServices = append(enabledServices, "ovnnb", "ovnsb", "ovn-northd")
			}

			if srv.Service == "switch" {
				enabledServices = append(enabledServices, "ovn-controller")
			}
		}
		return nil
	})

	if err != nil {
		enabledServices = nil
		err = fmt.Errorf("failed to lookup local services eligible for certificate refresh: %s", err)
	}

	// We always want a client certificate
	enabledServices = append(enabledServices, "client")

	return enabledServices, err
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
