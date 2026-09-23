package inspect

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/canonical/microcluster/v3/microcluster"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"

	"github.com/canonical/microovn/microovn/api/types"
)

const collectionTimeout = 30 * time.Second

type Input struct {
	ExecutionContext      types.InspectionExecutionContext
	Services              types.Services
	Members               []microTypes.ClusterMember
	Snapshots             map[string]types.InspectionNodeSnapshot
	Schemas               map[string]types.InspectionSchemaEvidence
	Communication         types.InspectionCommunicationEvidence
	Network               types.InspectionNetworkProbe
	CollectionErrors      []types.InspectionCollectionError
	DesiredStateAvailable bool
}

type Check interface {
	Name() string
	Run(context.Context, Input) []types.InspectionResult
}

func DefaultChecks() []Check {
	return []Check{
		TopologyCheck{},
		ServiceRuntimeCheck{},
		DatabaseCheck{},
		EnvironmentCheck{},
		NetworkCheck{},
	}
}

type CollectInputFunc func(context.Context, *microcluster.MicroCluster, microTypes.Client) (Input, error)

type Dependencies struct {
	Cluster      *microcluster.MicroCluster
	Client       microTypes.Client
	Checks       []Check
	CollectInput CollectInputFunc
}

type memberClient struct {
	member microTypes.ClusterMember
	client microTypes.Client
}

func Orchestrate(ctx context.Context, deps Dependencies) (types.InspectionReport, error) {
	ctx, cancel := context.WithTimeout(ctx, collectionTimeout)
	defer cancel()

	collect := deps.CollectInput
	if collect == nil {
		collect = collectInput
	}

	checks := deps.Checks
	if checks == nil {
		checks = DefaultChecks()
	}

	input, err := collect(ctx, deps.Cluster, deps.Client)
	if err != nil {
		return types.InspectionReport{}, err
	}
	return buildReport(ctx, time.Now().UTC(), input, checks), nil
}

func summarizeSchema(evidence types.InspectionSchemaEvidence) types.InspectionDatabaseStatus {
	summary := types.InspectionDatabaseStatus{
		ActiveSchemaVersion: evidence.ActiveVersion,
		ExpectedMembers:     len(evidence.Members),
	}

	mismatch := false
	incomplete := evidence.Database == "" ||
		evidence.ActiveVersion == "" ||
		evidence.ActiveError != ""

	for _, member := range evidence.Members {
		if member.Version != "" || member.Unsupported {
			summary.RespondingMembers++
		}

		if member.Version != "" {
			if evidence.ActiveVersion != "" &&
				member.Version != evidence.ActiveVersion {
				mismatch = true
			}
		}

		if member.Version == "" || member.Unsupported || member.Error != "" {
			incomplete = true
		}
	}

	switch {
	case mismatch:
		summary.Status = types.InspectionStatusFail
		summary.Message = "schema versions do not match"
	case incomplete:
		summary.Status = types.InspectionStatusUnknown
		summary.Message = "schema evidence is incomplete"
	default:
		summary.Status = types.InspectionStatusPass
	}

	return summary
}

func summarizeCommunication(
	evidence types.InspectionCommunicationEvidence,
) types.InspectionCommunicationSummary {
	summary := types.InspectionCommunicationSummary{
		NBCfg: evidence.NBCfg,
		SBCfg: evidence.SBCfg,
	}

	switch {
	case evidence.NBReachable != nil && !*evidence.NBReachable:
		summary.Status = types.InspectionStatusFail
		summary.Message = "Northbound database is unreachable"

	case evidence.SBReachable != nil && !*evidence.SBReachable:
		summary.Status = types.InspectionStatusFail
		summary.Message = "Southbound database is unreachable"

	case evidence.NBReachable == nil ||
		evidence.SBReachable == nil ||
		evidence.Converged == nil:
		summary.Status = types.InspectionStatusUnknown
		summary.Message = evidence.CollectionError
		if summary.Message == "" {
			summary.Message = "database communication could not be determined"
		}

	case !*evidence.Converged:
		summary.Status = types.InspectionStatusWarning
		summary.Message = "Southbound database has not converged"

	default:
		summary.Status = types.InspectionStatusPass
	}

	return summary
}

func summarize(results []types.InspectionResult) types.InspectionSummary {
	var summary types.InspectionSummary

	for _, result := range results {
		switch result.Status {
		case types.InspectionStatusPass:
			summary.Counts.Pass++
		case types.InspectionStatusWarning:
			summary.Counts.Warning++
		case types.InspectionStatusFail:
			summary.Counts.Fail++
		case types.InspectionStatusUnknown:
			summary.Counts.Unknown++
		}
	}

	switch {
	case summary.Counts.Fail > 0:
		summary.Status = types.InspectionStatusFail
	case summary.Counts.Unknown > 0:
		summary.Status = types.InspectionStatusUnknown
	case summary.Counts.Warning > 0:
		summary.Status = types.InspectionStatusWarning
	default:
		summary.Status = types.InspectionStatusPass
	}

	return summary
}

func buildReport(ctx context.Context, now time.Time, input Input, checks []Check) types.InspectionReport {
	var results []types.InspectionResult

	if !input.ExecutionContext.Authoritative {
		results = append(results, types.InspectionResult{
			ID:       "execution-authority",
			Category: "cluster",
			Status:   types.InspectionStatusWarning,
			Summary: fmt.Sprintf(
				"Inspection on %s is not authoritative",
				input.ExecutionContext.MemberRole,
			),
		})
	}

	for _, check := range checks {
		results = append(results, check.Run(ctx, input)...)
	}

	for index := range results {
		slices.SortFunc(
			results[index].Details,
			func(a, b types.InspectionDetail) int {
				if result := cmp.Compare(a.Node, b.Node); result != 0 {
					return result
				}
				return cmp.Compare(a.ID, b.ID)
			},
		)
	}

	databaseSummary := types.InspectionDatabaseSummary{
		Northbound:    summarizeSchema(input.Schemas["nb"]),
		Southbound:    summarizeSchema(input.Schemas["sb"]),
		Communication: summarizeCommunication(input.Communication),
	}
	if !input.ExecutionContext.Authoritative {
		databaseSummary.Northbound.Status = types.InspectionStatusUnknown
		databaseSummary.Northbound.Message = "cluster schema health is unavailable from a non-voting member"
		databaseSummary.Southbound.Status = types.InspectionStatusUnknown
		databaseSummary.Southbound.Message = "cluster schema health is unavailable from a non-voting member"
		databaseSummary.Communication.Status = types.InspectionStatusUnknown
		databaseSummary.Communication.Message = "cluster database communication is unavailable from a non-voting member"
	}

	return types.InspectionReport{
		SchemaVersion:    types.InspectionSchemaVersion,
		Timestamp:        now.UTC(),
		ExecutionContext: input.ExecutionContext,
		Summary:          summarize(results),
		DatabaseSummary:  databaseSummary,
		Results:          results,
	}
}
