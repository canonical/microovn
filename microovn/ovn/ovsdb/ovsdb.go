package ovsdb

import (
	"fmt"
	"strings"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/microcluster/state"

	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
)

// ExpectedOvsdbSchemaVersion returns version of the database schema that was shipped with current OVN/OVS
// packages. This value can be used to check whether current OVN/OVS processes are using up-to-date database
// schemas.
func ExpectedOvsdbSchemaVersion(s *state.State, dbSpec *ovnCmd.OvsdbSpec) (string, error) {
	targetDbVersion, err := shared.RunCommandContext(
		s.Context,
		"ovsdb-tool",
		"schema-version",
		dbSpec.Schema,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get DB schema version from file '%s': '%s'", dbSpec.Schema, err)
	}

	return strings.TrimSpace(targetDbVersion), nil
}
