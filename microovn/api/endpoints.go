// Package api provides the REST API endpoints.
package api

import (
	"github.com/canonical/microcluster/rest"
	"github.com/canonical/microovn/microovn/api/ovsdb"

	"github.com/canonical/microovn/microovn/api/certificates"
	"github.com/canonical/microovn/microovn/api/services"
	"github.com/canonical/microovn/microovn/api/types"
)

// Server is an extension to the default microcluster server, which serves the supplied endpoints over "/1.0"
var Server = rest.Server{
	CoreAPI:   true,
	ServeUnix: true,
	Resources: []rest.Resources{
		{
			PathPrefix: types.APIVersion,
			Endpoints: []rest.Endpoint{
				services.ListCmd,
				certificates.IssueCertificatesEndpoint,
				certificates.IssueCertificatesAllEndpoint,
				certificates.RegenerateCaEndpoint,
				ovsdb.ActiveSchemaVersion,
				ovsdb.AllExpectedSchemaVersions,
				ovsdb.ExpectedSchemaVersion,
			},
		},
	},
}
