package ovsdb

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/canonical/lxd/lxd/response"
	"github.com/canonical/lxd/shared/logger"
	"github.com/canonical/microcluster/rest"
	"github.com/canonical/microcluster/state"
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

// getExpectedSchemaVersion implements GET method for /1.0/ovsdb/schema/<db-name>/expected. It returns
// expected schema version of the specified database on this MicroOVN node. Expected schema version is
// determined by looking at the schema file that is bundled with OVN/OVS packages.
// URL variable <db-name> is expected to be one of the keys from supportedDBs.
func getExpectedSchemaVersion(s *state.State, r *http.Request) response.Response {
	requestedDB, err := url.PathUnescape(mux.Vars(r)["db"])
	if err != nil {
		logger.Errorf("failed to parse requested DB name from url '%s'", r.URL)
		return response.ErrorResponse(500, "Internal Server Error")
	}
	requestedDB = strings.ToLower(requestedDB)

	dbType, ok := supportedDBs[requestedDB]

	if !ok {
		return response.BadRequest(fmt.Errorf("DB '%s' not supported", requestedDB))
	}

	dbSpec, err := ovnCmd.NewOvsdbSpec(dbType)
	if err != nil {
		logger.Errorf("%s", err)
		return response.ErrorResponse(500, "Internal Server Error")
	}

	expectedVersion, err := ovsdb.ExpectedOvsdbSchemaVersion(s, dbSpec)
	if err != nil {
		logger.Errorf("failed to get expected OVSDB schema version for '%s' database: %s", dbSpec.FriendlyName, err)
		return response.ErrorResponse(500, "Internal Server Error")
	}

	return response.SyncResponse(true, expectedVersion)
}
