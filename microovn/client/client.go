// Package client provides a full Go API client.
package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/canonical/lxd/shared/api"
	"github.com/canonical/microcluster/client"

	"github.com/canonical/microovn/microovn/api/types"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
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

// GetExpectedOvsdbSchemaVersion queries given MicroOVN node and returns an expected schema version for the specified
// database. This is not necessarily the schema version that's being used by currently running OVN/OVS processes on the
// node. Rather it's a version of a schema that was supplied with currently installed OVN/OVS packages on the node.
// A discrepancy between these two can occur when MicroOVN gets upgraded, but cluster-wide schema upgrade was not
// triggered, or completed, yet.
func GetExpectedOvsdbSchemaVersion(ctx context.Context, c *client.Client, dbSpec *ovnCmd.OvsdbSpec) (string, types.OvsdbSchemaFetchError) {
	return getOvsdbSchemaVersion(ctx, c, dbSpec, "expected")
}

// GetAllExpectedOvsdbSchemaVersions returns types.OvsdbSchemaReport. It is a list containing every node of the MicroOVN
// deployment and for each node it contains node's Hostname, a version of the OVSDB schema expected on that node and
// whether there were any errors while fetching information from that node.
func GetAllExpectedOvsdbSchemaVersions(ctx context.Context, c *client.Client, dbSpec *ovnCmd.OvsdbSpec) (types.OvsdbSchemaReport, error) {
	var response types.OvsdbSchemaReport

	queryCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	err := c.Query(queryCtx, "GET", api.NewURL().Path("ovsdb", "schema", dbSpec.ShortName, "expected", "all"), nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get expected ovsdb schema versions from cluster: %w", err)
	}

	return response, nil
}

// GetActiveOvsdbSchemaVersion queries MicroOVN cluster for a version of the schema that's currently used by a database
// specified by the "dbSpec" argument.
func GetActiveOvsdbSchemaVersion(ctx context.Context, c *client.Client, dbSpec *ovnCmd.OvsdbSpec) (string, types.OvsdbSchemaFetchError) {
	return getOvsdbSchemaVersion(ctx, c, dbSpec, "active")
}

// getOvsdbSchemaVersion is a general function that is used to fetch OVSDB schema version via MicroOVN API. It targets
// /1.0/ovsdb/schema/<db-name>/<target> endpoints, where <db-name> is ovnCmd.OvsdbSpec.ShortName and <target> is
// either "active", "expected", or other variations that MicroOVN API supports.
func getOvsdbSchemaVersion(ctx context.Context, c *client.Client, dbSpec *ovnCmd.OvsdbSpec, target string) (string, types.OvsdbSchemaFetchError) {
	var response string

	queryCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	err := c.Query(queryCtx, "GET", api.NewURL().Path("ovsdb", "schema", dbSpec.ShortName, target), nil, &response)

	if err != nil {
		var errorStatus api.StatusError
		errIdentified := errors.As(err, &errorStatus)
		if errIdentified && errorStatus.Status() == http.StatusNotFound {
			return "", types.OvsdbSchemaFetchErrorNotSupported
		} else {
			return "", types.OvsdbSchemaFetchErrorGeneric
		}
	}

	return response, types.OvsdbSchemaFetchErrorNone
}
