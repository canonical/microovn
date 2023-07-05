// Package client provides a full Go API client.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/microcluster/client"

	"github.com/canonical/microovn/microovn/api/types"
)

// GetServices returns the list of configured OVN services.
func GetServices(ctx context.Context, c *client.Client) (types.Services, error) {
	queryCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	services := types.Services{}

	err := c.Query(queryCtx, "GET", api.NewURL().Path("services"), nil, &services)
	if err != nil {
		return nil, fmt.Errorf("Failed listing services: %w", err)
	}

	return services, nil
}

// ReissueCertificate sends request to local MicroOVN cluster member to re-issue new certificate for
// selected service.
func ReissueCertificate(ctx context.Context, c *client.Client, serviceName string) (types.IssueCertificateResponse, error) {
	queryCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	response := types.IssueCertificateResponse{}
	err := c.Query(queryCtx, "PUT", api.NewURL().Path("certificates", serviceName), nil, &response)
	if err != nil {
		return response, fmt.Errorf("failed to reissue certificate: %w", err)
	}

	return response, nil
}

// ReissueAllCertificate sends request to local MicroOVN cluster member to re-issue new certificates for every
// enabled OVN service present.
func ReissueAllCertificate(ctx context.Context, c *client.Client) (types.IssueCertificateResponse, error) {
	queryCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	response := types.IssueCertificateResponse{}
	err := c.Query(queryCtx, "PUT", api.NewURL().Path("certificates"), nil, &response)
	if err != nil {
		return response, fmt.Errorf("failed to reissue certificate: %w", err)
	}

	return response, nil
}

// RegenerateCA sends request to completely rebuild the OVN PKI. It causes new CA certificate to be issued and shared
// between MicroOVN cluster members, and it triggers re-issue of all OVN service certificates on all cluster members.
func RegenerateCA(ctx context.Context, c *client.Client) (types.RegenerateCaResponse, error) {
	queryCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	response := types.NewRegenerateCaResponse()

	err := c.Query(queryCtx, "PUT", api.NewURL().Path("ca"), nil, &response)
	if err != nil {
		return *response, fmt.Errorf("failed to generate new CA: %w", err)
	}

	return *response, nil

}
