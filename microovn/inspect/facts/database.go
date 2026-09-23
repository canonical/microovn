package facts

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/canonical/microcluster/v3/state"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/ovn/cmd"
	ovnOvsdb "github.com/canonical/microovn/microovn/ovn/ovsdb"
)

const (
	probePollInterval = 500 * time.Millisecond
	probeDeadline     = 5 * time.Second
)

func CollectDatabaseProbe(
	ctx context.Context,
	s state.State,
	scope types.InspectionScope,
) (types.InspectionDatabaseProbe, error) {
	var readNB, readSB dbCfgReader
	switch scope {
	case types.InspectionScopeCluster:
		readNB, readSB = dbCfgReadersCluster(s)
	case types.InspectionScopeLocal:
		readNB, readSB = dbCfgReadersLocal(s)
	default:
		return types.InspectionDatabaseProbe{}, fmt.Errorf("invalid scope: %s", scope)
	}

	var schemas <-chan []types.InspectionSchemaEvidence
	if scope == types.InspectionScopeLocal {
		result := make(chan []types.InspectionSchemaEvidence, 1)
		schemas = result
		go func() {
			result <- collectSchemas(ctx, s)
		}()
	}

	probe := types.InspectionDatabaseProbe{
		Scope: scope,
		Communication: probeDatabaseCommunication(
			ctx,
			readNB,
			readSB,
			probePollInterval,
			probeDeadline,
		),
	}

	if schemas != nil {
		probe.Schemas = <-schemas
	}

	return probe, nil
}

type dbCfgParseError struct {
	database string
}

func (e dbCfgParseError) Error() string {
	return fmt.Sprintf("invalid %s nb_cfg value", e.database)
}

type dbCfgReader func(context.Context) (int64, error)

func dbCfgReadersCluster(s state.State) (dbCfgReader, dbCfgReader) {
	return dbCfgReaders(s, cmd.NBCtlCluster, cmd.SBCtlCluster)
}

func dbCfgReadersLocal(s state.State) (dbCfgReader, dbCfgReader) {
	return dbCfgReaders(s, cmd.NBCtl, cmd.SBCtl)
}

func probeDatabaseCommunication(
	ctx context.Context,
	readNB dbCfgReader,
	readSB dbCfgReader,
	pollInterval time.Duration,
	deadline time.Duration,
) types.InspectionCommunicationEvidence {
	var evidence types.InspectionCommunicationEvidence

	nbCfg, err := readNB(ctx)
	if err != nil {
		var nbReachable *bool
		var parseError dbCfgParseError
		if errors.As(err, &parseError) {
			nbReachable = boolPtr(true)
		} else if ctx.Err() == nil {
			nbReachable = boolPtr(false)
		}
		collectionError := fmt.Sprintf("failed to read NB configuration: %v", err)
		if ctx.Err() != nil {
			collectionError = fmt.Sprintf("failed to read NB configuration: %v", ctx.Err())
		}
		return types.InspectionCommunicationEvidence{
			NBReachable:     nbReachable,
			CollectionError: collectionError,
		}
	}

	evidence.NBCfg = &nbCfg
	evidence.NBReachable = boolPtr(true)

	sbCfg, converged, err := pollSBUntilConverged(ctx, readSB, nbCfg, pollInterval, deadline)
	if err != nil {
		var sbReachable *bool
		var parseError dbCfgParseError
		if sbCfg != nil || errors.As(err, &parseError) {
			sbReachable = boolPtr(true)
		} else if ctx.Err() == nil && !errors.Is(err, context.DeadlineExceeded) {
			sbReachable = boolPtr(false)
		}
		collectionError := fmt.Sprintf("failed to read SB configuration: %v", err)
		if ctx.Err() != nil {
			collectionError = fmt.Sprintf("failed to read SB configuration: %v", ctx.Err())
		}
		return types.InspectionCommunicationEvidence{
			NBCfg:           evidence.NBCfg,
			SBCfg:           sbCfg,
			NBReachable:     evidence.NBReachable,
			SBReachable:     sbReachable,
			Converged:       converged,
			CollectionError: collectionError,
		}
	}

	evidence.SBCfg = sbCfg
	evidence.SBReachable = boolPtr(true)
	evidence.Converged = converged
	return evidence
}

func collectSchemas(ctx context.Context, s state.State) []types.InspectionSchemaEvidence {
	evidence := make([]types.InspectionSchemaEvidence, 0, 2)

	for _, dbType := range []cmd.OvsdbType{
		cmd.OvsdbTypeNBLocal,
		cmd.OvsdbTypeSBLocal,
	} {
		dbSpec, err := cmd.NewOvsdbSpec(dbType)
		if err != nil {
			continue
		}

		schema := types.InspectionSchemaEvidence{
			Database: dbSpec.ShortName,
			Members: []types.InspectionSchemaMemberEvidence{
				{Node: s.Name()},
			},
		}

		active, err := cmd.OvsdbClient(
			ctx, s, dbSpec,
			1, 1,
			"get-schema-version",
			dbSpec.SocketURL,
		)
		if err != nil {
			schema.ActiveError = "active schema version could not be collected"
		} else {
			schema.ActiveVersion = strings.TrimSpace(active)
		}

		expected, err := ovnOvsdb.ExpectedOvsdbSchemaVersion(ctx, s, dbSpec)
		if err != nil {
			schema.Members[0].Error = "expected schema version could not be collected"
		} else {
			schema.Members[0].Version = expected
		}

		evidence = append(evidence, schema)
	}

	return evidence
}

func parseDBCfg(database string, output string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
	if err != nil {
		return 0, dbCfgParseError{database: database}
	}

	return value, nil
}

func dbCfgReaders(
	s state.State,
	nbCtl func(context.Context, state.State, int, ...string) (string, error),
	sbCtl func(context.Context, state.State, int, ...string) (string, error),
) (dbCfgReader, dbCfgReader) {
	var nbCfgArgs = []string{
		"--timeout", "1",
		"--format=csv", "--no-headings", "--data=bare",
		"get", "NB_Global", ".", "nb_cfg",
	}

	var sbCfgArgs = []string{
		"--timeout", "1",
		"--format=csv", "--no-headings", "--data=bare",
		"get", "SB_Global", ".", "nb_cfg",
	}

	return func(ctx context.Context) (int64, error) {
			output, err := nbCtl(ctx, s, 1, nbCfgArgs...)
			if err != nil {
				return 0, err
			}
			return parseDBCfg("NB", output)
		}, func(ctx context.Context) (int64, error) {
			output, err := sbCtl(ctx, s, 1, sbCfgArgs...)
			if err != nil {
				return 0, err
			}
			return parseDBCfg("SB", output)
		}
}
