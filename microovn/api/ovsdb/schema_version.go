package ovsdb

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/client"
	"github.com/canonical/microcluster/rest"
	"github.com/canonical/microcluster/state"
	"github.com/canonical/microovn/microovn/api/types"
	microovnClient "github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/node"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/gorilla/mux"

	"github.com/canonical/microovn/microovn/ovn/ovsdb"
)

// supportedDBs contains all database types that support reporting of
// a schema version.
var supportedDBs = map[string]ovnCmd.OvsdbType{
	"sb":     ovnCmd.OvsdbTypeSBLocal,
	"nb":     ovnCmd.OvsdbTypeNBLocal,
	"switch": ovnCmd.OvsdbTypeSwitchLocal,
}

// ExpectedSchemaVersion defines endpoints for /1.0/ovsdb/schema/<db-name>/expected
var ExpectedSchemaVersion = rest.Endpoint{
	Path: "ovsdb/schema/{db}/expected",
	Get:  rest.EndpointAction{Handler: getExpectedSchemaVersion, AllowUntrusted: false, ProxyTarget: false},
}

// AllExpectedSchemaVersions defines endpoints for /1.0/ovsdb/schema/<db-name>/expected/all
var AllExpectedSchemaVersions = rest.Endpoint{
	Path: "ovsdb/schema/{db}/expected/all",
	Get:  rest.EndpointAction{Handler: getAllExpectedSchemaVersions, AllowUntrusted: false, ProxyTarget: false},
}

// ActiveSchemaVersion defines endpoints for /1.0/ovsdb/schema/<db-name>/active
var ActiveSchemaVersion = rest.Endpoint{
	Path: "ovsdb/schema/{db}/active",
	Get:  rest.EndpointAction{Handler: getActiveSchemaVersion, AllowUntrusted: false, ProxyTarget: false},
}

// getExpectedSchemaVersion implements GET method for /1.0/ovsdb/schema/<db-name>/expected. It returns
// expected schema version of the specified database on this MicroOVN node. Expected schema version is
// determined by looking at the schema file that is bundled with OVN/OVS packages.
// URL variable <db-name> is expected to be one of the keys from supportedDBs.
func getExpectedSchemaVersion(s *state.State, r *http.Request) response.Response {
	dbSpec, errResponse := parseDbSpec(r)
	if errResponse != nil {
		return errResponse
	}

	expectedVersion, err := ovsdb.ExpectedOvsdbSchemaVersion(s, dbSpec)
	if err != nil {
		logger.Errorf("failed to get expected OVSDB schema version for '%s' database: %s", dbSpec.FriendlyName, err)
		return response.ErrorResponse(500, "Internal Server Error")
	}

	return response.SyncResponse(true, expectedVersion)
}

// getAllExpectedSchemaVersions implements GET method for /1.0/ovsdb/schema/<db-name>/expected/all.
// It returns expected schema version for the given database from each node in the deployment.
// The response is in the format of types.OvsdbSchemaReport
func getAllExpectedSchemaVersions(s *state.State, r *http.Request) response.Response {
	dbSpec, errResponse := parseDbSpec(r)
	if errResponse != nil {
		return errResponse
	}

	// Get local expected schema version and store it in the final result
	localExpectedVersion, err := ovsdb.ExpectedOvsdbSchemaVersion(s, dbSpec)
	if err != nil {
		logger.Errorf("failed to get expected OVSDB schema version for '%s' database: %s", dbSpec.FriendlyName, err)
		return response.ErrorResponse(500, "Internal Server Error")
	}

	responseData := types.OvsdbSchemaReport{
		types.OvsdbSchemaVersionResult{
			SchemaVersion: localExpectedVersion,
			Host:          s.Address().Hostname(),
			Error:         types.OvsdbSchemaFetchErrorNone,
		},
	}

	// Get clients for each member in the cluster
	clusterClient, err := s.Cluster(r)
	if err != nil {
		logger.Errorf("Failed to get a client for every cluster member: %s", err)
		return response.ErrorResponse(500, "Internal Server Error")
	}

	// Fetch expected schema versions from each cluster member.
	_ = clusterClient.Query(s.Context, true, func(ctx context.Context, c *client.Client) error {
		clientUrl := c.URL()
		logger.Debugf("Fetching expected OVN %s schema version from '%s'", dbSpec.FriendlyName, clientUrl.String())
		nodeStatus := types.OvsdbSchemaVersionResult{Host: clientUrl.Hostname()}

		result, responseSuccess := microovnClient.GetExpectedOvsdbSchemaVersion(ctx, c, dbSpec)
		nodeStatus.Error = responseSuccess
		nodeStatus.SchemaVersion = result

		responseData = append(responseData, nodeStatus)
		return nil
	})

	return response.SyncResponse(true, &responseData)
}

// getActiveSchemaVersion implements GET method for /1.0/ovsdb/schema/<db-name>/active. It returns
// currently active schema version of a database specified by <db-name>.
// URL variable <db-name> is expected to be one of the keys from supportedDBs.
//
// If the node receives request for Northbound or Southbound DB schema version, but the not is not running
// these central services, the request will be forwarded to a node that does run them.
func getActiveSchemaVersion(s *state.State, r *http.Request) response.Response {
	hasCentral, err := node.HasServiceActive(s, "central")
	if err != nil {
		logger.Errorf("failed to check if central is active on this node: %s", err)
		return response.ErrorResponse(500, "Internal Server Error")
	}

	dbSpec, errResponse := parseDbSpec(r)
	if errResponse != nil {
		return errResponse
	}

	// If Northbound or Southbound database was requested, but we don't run "central" services,
	// forward this request to a node that does.
	if dbSpec.IsCentral && !hasCentral {
		logger.Info("This node does not run 'central' service. Request will be forwarded.")
		return forwardActiveSchemaVersion(s, r, dbSpec)
	}

	activeSchema, err := ovnCmd.OvsdbClient(s, dbSpec, 10, 30, "get-schema-version", dbSpec.SocketURL)
	if err != nil {
		logger.Errorf("failed to get active schema version for '%s' database: %s", dbSpec.FriendlyName, err)
		return response.ErrorResponse(500, "Internal Server Error")
	}

	return response.SyncResponse(true, strings.TrimSpace(activeSchema))
}

// forwardActiveSchemaVersion forwards request to get active OVSDB schema version to a host that runs "central"
// services.
// Each host that is registered with "central" service is queried until one of the returns non-error response. First
// successful response is returned to the caller.
// If none of the "central" nodes return non-error message, this function returns response.ErrorResponse with code 500.
func forwardActiveSchemaVersion(s *state.State, r *http.Request, dbSpec *ovnCmd.OvsdbSpec) response.Response {
	centralNodes, err := node.FindService(s, "central")
	if err != nil {
		logger.Errorf("failed to find central node: %s", err)
		return response.ErrorResponse(500, "Internal Server Error")
	}

	clusterClients, err := s.Cluster(r)

	for _, client_ := range clusterClients {
		for _, node_ := range centralNodes {
			clientUrl := client_.URL()
			clientAddr := fmt.Sprintf("%s:%s", clientUrl.Hostname(), clientUrl.Port())
			if clientAddr == node_.Address {
				logger.Infof(
					"Forwarding request '%s' for active %s schema to %s",
					r.URL,
					dbSpec.FriendlyName,
					node_.Name,
				)
				result, err := microovnClient.GetActiveOvsdbSchemaVersion(s.Context, &client_, dbSpec)
				if err != types.OvsdbSchemaFetchErrorNone {
					logger.Errorf(
						"Failed to forward request for active %s schema version to node %s",
						dbSpec.FriendlyName,
						node_.Name,
					)
				} else {
					return response.SyncResponse(true, &result)
				}
			}
		}
	}

	logger.Error("None of the central nodes responded to the forwarded query")
	return response.ErrorResponse(500, "Internal Server Error")
}

// parseDbSpec is a helper function that returns OvsdbSpec based on the database name
// specified in the request's URL variable.
//
// For example: If request "r" has URL /schema/sb/active, this function  returns OvsdbSpec based on OvsdbTypeSBLocal.
func parseDbSpec(r *http.Request) (*ovnCmd.OvsdbSpec, response.Response) {
	requestedDB, err := url.PathUnescape(mux.Vars(r)["db"])
	if err != nil {
		logger.Errorf("failed to parse requested DB name from url '%s'", r.URL)
		return nil, response.ErrorResponse(500, "Internal Server Error")
	}
	requestedDB = strings.ToLower(requestedDB)

	dbType, ok := supportedDBs[requestedDB]

	if !ok {
		return nil, response.BadRequest(fmt.Errorf("DB '%s' not supported", requestedDB))
	}

	dbSpec, err := ovnCmd.NewOvsdbSpec(dbType)
	if err != nil {
		logger.Errorf("%s", err)
		return nil, response.ErrorResponse(500, "Internal Server Error")
	}
	return dbSpec, nil
}
