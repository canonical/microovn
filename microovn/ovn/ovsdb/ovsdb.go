// Package ovsdb provides functions for handling OVSDB schema.
package ovsdb

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/logger"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/canonical/microcluster/v3/state"

	"github.com/canonical/microovn/microovn/api/types"
	microovnClient "github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/node"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
)

// Constants below should be used when implementing retry algorithms with increasingly longer back-offs between
// attempts
const (
	backOffMsInitial  = 100  // amount of milliseconds to wait before first retry
	backOffMsMax      = 2000 // Maximum amount of milliseconds to wait between retry attempts
	backOffMultiplier = 2    // multiplier for increasingly longer back-offs after each retry attempt
)

// schemaStatus structure encapsulates information about schema version
// status of a specific OVSDB.
type schemaStatus struct {
	LocalVersion    string // Version of the schema that's currently running
	TargetVersion   string // Version of the schema that's expected to be running. (based on the schema supplied by the OVN/OVS package)
	UpgradeRequired bool   // Whether an OVSDB schema upgrade is required
}

// ExpectedOvsdbSchemaVersion returns version of the database schema that was shipped with current OVN/OVS
// packages. This value can be used to check whether current OVN/OVS processes are using up-to-date database
// schemas.
func ExpectedOvsdbSchemaVersion(ctx context.Context, _ state.State, dbSpec *ovnCmd.OvsdbSpec) (string, error) {
	targetDbVersion, err := shared.RunCommandContext(
		ctx,
		"ovsdb-tool",
		"schema-version",
		dbSpec.Schema,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get DB schema version from file '%s': '%s'", dbSpec.Schema, err)
	}

	return strings.TrimSpace(targetDbVersion), nil
}

// getLiveSchemaStatus returns information about schema status of the database specified by the "dbSpec" argument.
// For more information about the returned value, see: schemaStatus.
func getLiveSchemaStatus(ctx context.Context, s state.State, dbSpec *ovnCmd.OvsdbSpec) (schemaStatus, error) {
	var status schemaStatus

	localDbVersion, err := ovnCmd.OvsdbClient(
		ctx,
		s,
		dbSpec,
		10,
		5,
		"get-schema-version",
		dbSpec.SocketURL,
		dbSpec.Name,
	)

	if err != nil {
		return schemaStatus{}, fmt.Errorf("failed to get local DB schema version: '%s'", err.Error())
	}
	localDbVersion = strings.TrimSpace(localDbVersion)

	targetDbVersion, err := ExpectedOvsdbSchemaVersion(ctx, s, dbSpec)
	if err != nil {
		return schemaStatus{}, fmt.Errorf("failed to get expected DB schema version: '%s'", err.Error())
	}

	_, err = shared.RunCommandContext(
		ctx,
		"ovsdb-tool",
		"compare-versions",
		localDbVersion,
		"==",
		targetDbVersion,
	)

	msg := fmt.Sprintf(
		"Curently running %s DB schema version: %s; Expected: %s;",
		dbSpec.FriendlyName,
		localDbVersion,
		targetDbVersion,
	)
	if err != nil {
		msg = fmt.Sprintf("%s Upgrade required", msg)
	} else {
		msg = fmt.Sprintf("%s No upgrade required", msg)
	}

	logger.Debug(msg)

	status.UpgradeRequired = err != nil
	status.LocalVersion = localDbVersion
	status.TargetVersion = targetDbVersion

	return status, nil
}

// isNodeUpgradeLeader returns "true" if current node is a designated leader for performing OVN database schema
// upgrade.
// It does not matter which node triggers the schema upgrade as long as it's running a OVN central service. Therefore,
// a node with "central" service active and the lowest ID is chosen. A node ID is an internal value of a MicroOVN, and
// it's kept consistent across the cluster via the underlying MicroCluster library.
func isNodeUpgradeLeader(ctx context.Context, s state.State) (bool, error) {
	membersWithCentral, err := node.FindService(ctx, s, "central")
	if err != nil {
		return false, fmt.Errorf("failed to determine which member is OVSDB cluster upgrade leader: '%s'", err)
	}

	sort.Slice(membersWithCentral, func(i, j int) bool {
		return membersWithCentral[i].ID < membersWithCentral[j].ID
	})
	upgradeLeader := membersWithCentral[0]

	return s.Name() == upgradeLeader.Name, nil
}

// isClusterUpgradeReady returns "true" if expected schema version of a database, on every cluster member, matches an
// expected schema version on current member.
// This function scans every cluster member, regardless of whether they are running "central" services or not. This
// ensures that even cluster members that are running on "chassis" service are prepared for the schema upgrade.
func isClusterUpgradeReady(ctx context.Context, s state.State, dbSpec *ovnCmd.OvsdbSpec, targetVersion string) (bool, error) {
	clusterClient, err := s.Connect().Cluster(false)
	if err != nil {
		return false, fmt.Errorf("failed to get a client for every cluster member: %w", err)
	}

	results := map[string]string{}

	// Gather expected schema version from every member in the cluster via their API.
	err = clusterClient.Query(ctx, true, func(ctx context.Context, c microTypes.Client) error {
		clientURL := c.URL()
		clientURLString := clientURL.String()
		logger.Debugf("Requesting OVSDB %s schema status from '%s'", dbSpec.FriendlyName, clientURLString)
		result, errType := microovnClient.GetExpectedOvsdbSchemaVersion(ctx, c, dbSpec)
		if errType != types.OvsdbSchemaFetchErrorNone {
			var errMsg string
			if errType == types.OvsdbSchemaFetchErrorNotSupported {
				errMsg = "API endpoint not found. Perhaps this MicroOVN requires upgrade?"
			} else {
				errMsg = "API endpoint returned error. Try looking into logs on the target member."
			}
			logger.Errorf(
				"Failed to get OVN %s DB schema status from '%s': %s",
				dbSpec.FriendlyName,
				clientURLString,
				errMsg,
			)
			return fmt.Errorf("failed to contact %s", clientURLString)
		}
		results[clientURLString] = result
		return nil
	})

	if err != nil {
		return false, fmt.Errorf("failed to assess overall cluster readiness for OVN cluster DB upgrade: '%s'", err)
	}

	// Match expected schema version from cluster members against our expected version. Only if all values match,
	// return "true", indicating that the cluster is ready for the schema upgrade.
	clusterReady := true
	for host, status := range results {
		if status == targetVersion {
			logger.Infof("Host at '%s' has OVN %s DB ready for upgrade", host, dbSpec.FriendlyName)
		} else {
			logger.Infof("Host at '%s' does not have OVN %s DB ready for update", host, dbSpec.FriendlyName)
			clusterReady = false
		}
	}

	return clusterReady, nil
}

// UpgradeCentralDB checks whether a schema upgrade for the specified DB is required. If so, it triggers
// a coordinated approach to an OVSDB cluster schema upgrade.
// The function will wait in loop until every member in the cluster is ready for the schema upgrade. Once the cluster
// is ready, a designated cluster member will trigger a schema upgrade. Non-designated members will keep looping until,
// eventually, their expected schema version will match the currently running schema version.
//
// Note: This function supports upgrade only for the "central" databases (e.i. OVN_Northbound and OVN_Southbound). It
//
//	will return error if other database is specified in the 'dbType' argument.
//
// Note2: This function has no effect on a node that does not run OVN "central" services, and it will return silently.
func UpgradeCentralDB(ctx context.Context, s state.State, dbType ovnCmd.OvsdbType) error {
	dbSpec, err := ovnCmd.NewOvsdbSpec(dbType)
	if err != nil {
		return fmt.Errorf("failed create DB specification: '%s'", err.Error())
	}

	if !dbSpec.IsCentral {
		return fmt.Errorf("MicroOVN handles upgrades only for Northbound and Southbound clustered DBs")
	}

	centralActive, err := node.HasServiceActive(ctx, s, "central")
	if err != nil {
		return fmt.Errorf("failed to query local services: %s", err)
	}

	if !centralActive {
		logger.Infof(
			"OVN %s database won't be upgraded from this node because 'central' service is not active",
			dbSpec.FriendlyName,
		)
		return nil
	}

	backOffMs := backOffMsInitial
	var dbStatus schemaStatus

	// Start of the loop that coordinates with other cluster members the OVSDB cluster schema upgrade.
	for {
		// Get current schema status
		dbStatus, err = getLiveSchemaStatus(ctx, s, dbSpec)
		if err != nil {
			logger.Warnf(
				"Error checking if OVN %s cluster needs upgrade: '%s' (retrying in %d ms)",
				dbSpec.FriendlyName,
				err,
				backOffMs,
			)
		} else {
			if !dbStatus.UpgradeRequired {
				// If upgrade is not required, break out of the loop
				logger.Infof("OVN %s DB is at expected version. No upgrade needed", dbSpec.FriendlyName)
				break
			}
			logger.Infof("OVN %s DB schema needs upgrade.", dbSpec.FriendlyName)

			// Check whether we are the designated node for triggering the schema upgrade
			upgradeLeader, err := isNodeUpgradeLeader(ctx, s)
			if err != nil {
				logger.Warnf(
					"Failed to determine if this host is OVN %s upgrade leader: '%s' (retrying in %d ms)",
					dbSpec.FriendlyName,
					err,
					backOffMs,
				)
			} else if upgradeLeader {

				// Leader verifies whether the cluster is ready for schema upgrade
				upgradeReady, err := isClusterUpgradeReady(ctx, s, dbSpec, dbStatus.TargetVersion)
				if err != nil {
					logger.Warnf(
						"OVN %s DB upgrade readycheck failed: '%s' (retrying in %d ms)",
						dbSpec.FriendlyName,
						err,
						backOffMs,
					)
				} else if upgradeReady {

					// If the cluster is ready, an upgrade is triggered. Otherwise, the loop continues.
					logger.Infof("Triggering OVN %s schema upgrade.", dbSpec.FriendlyName)
					_, err = ovnCmd.OvsdbClient(
						ctx,
						s,
						dbSpec,
						10,
						60,
						"convert",
						dbSpec.SocketURL,
						dbSpec.Schema,
					)
					return err
				}
			} else {
				// Nodes that are not designated to trigger the upgrade, continue looping. This ensures that in case
				// that cluster has to be altered during the upgrade, e.g. if the original leader has to be replaced,
				// a new member will become a designated leader, and it will trigger the upgrade.
				logger.Infof(
					"This host is not a designated to upgrade OVN %s DB schema. Rechecking schema status in %d",
					dbSpec.FriendlyName,
					backOffMs,
				)
			}
		}

		time.Sleep(time.Duration(backOffMs) * time.Millisecond)
		if backOffMs < backOffMsMax {
			backOffMs *= backOffMultiplier
		}

		if backOffMs > backOffMsMax {
			backOffMs = backOffMsMax
		}
	}

	return nil
}
