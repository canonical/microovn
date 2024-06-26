// Package api provides the REST API endpoints.
package api

import (
	"github.com/canonical/microcluster/rest"
	"github.com/canonical/microovn/microovn/api/ovsdb"

	"github.com/canonical/microovn/microovn/api/certificates"
)

// Endpoints is a global list of all API endpoints on the /1.0 endpoint of microovn.
var Endpoints = []rest.Endpoint{
	servicesCmd,
	certificates.IssueCertificatesEndpoint,
	certificates.IssueCertificatesAllEndpoint,
	certificates.RegenerateCaEndpoint,
	ovsdb.ActiveSchemaVersion,
	ovsdb.AllExpectedSchemaVersions,
	ovsdb.ExpectedSchemaVersion,
}
