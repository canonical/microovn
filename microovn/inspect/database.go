package inspect

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/canonical/microovn/microovn/api/types"
)

type DatabaseCheck struct{}

func (DatabaseCheck) Name() string {
	return "database"
}

func (DatabaseCheck) Run(_ context.Context, input Input) []types.InspectionResult {
	if !input.ExecutionContext.Authoritative {
		return []types.InspectionResult{nonAuthoritativeDatabaseResult(input)}
	}

	members := getSortedMemberNames(input)
	var results []types.InspectionResult
	for _, database := range []string{"nb", "sb"} {
		results = append(results, databaseSchemaResults(database, input.Schemas[database], members)...)
	}
	results = append(results, databaseCommunicationResult(input.Communication))
	if collectionError := databaseCollectionError(input); collectionError != "" {
		for index := range results {
			if results[index].Status == types.InspectionStatusUnknown {
				if results[index].CollectionError == "" {
					results[index].CollectionError = collectionError
				} else {
					results[index].CollectionError += "; " + collectionError
				}
				break
			}
		}
	}
	return results
}

func databaseSchemaResults(database string, evidence types.InspectionSchemaEvidence, members []string) []types.InspectionResult {
	byNode := make(map[string]types.InspectionSchemaMemberEvidence, len(evidence.Members))
	for _, member := range evidence.Members {
		byNode[member.Node] = member
	}

	var mismatches []types.InspectionDetail
	var unavailable []types.InspectionDetail
	var collectionErrors []string
	if evidence.ActiveVersion == "" || evidence.ActiveError != "" {
		summary := "Active schema version is unavailable"
		if evidence.ActiveError != "" {
			summary = evidence.ActiveError
			collectionErrors = append(collectionErrors, evidence.ActiveError)
		}
		unavailable = append(unavailable, types.InspectionDetail{
			ID:      "active-schema",
			Status:  types.InspectionStatusUnknown,
			Summary: summary,
		})
	}

	if len(members) == 0 {
		unavailable = append(unavailable, types.InspectionDetail{
			ID:      "schema-members",
			Status:  types.InspectionStatusUnknown,
			Summary: "Expected cluster members are unavailable",
		})
	}

	for _, node := range members {
		member, found := byNode[node]
		switch {
		case !found:
			unavailable = append(unavailable, schemaCoverageDetail(node, "Schema response is missing"))
		case member.Unsupported:
			unavailable = append(unavailable, schemaCoverageDetail(node, "Schema API is unsupported"))
		case member.Error != "":
			unavailable = append(unavailable, schemaCoverageDetail(node, member.Error))
			collectionErrors = append(collectionErrors, fmt.Sprintf("%s: %s", node, member.Error))
		case member.Version == "":
			unavailable = append(unavailable, schemaCoverageDetail(node, "Schema version is unavailable"))
		case evidence.ActiveVersion != "" && member.Version != evidence.ActiveVersion:
			mismatches = append(mismatches, types.InspectionDetail{
				Node:    node,
				ID:      "schema-version",
				Status:  types.InspectionStatusFail,
				Summary: "Expected schema version does not match the active version",
				Data: map[string]string{
					"active_version":   evidence.ActiveVersion,
					"expected_version": member.Version,
				},
			})
		}
	}

	var results []types.InspectionResult
	if len(mismatches) > 0 {
		results = append(results, types.InspectionResult{
			ID:          "database-schema-mismatch-" + database,
			Category:    "database",
			Status:      types.InspectionStatusFail,
			Summary:     strings.ToUpper(database) + " schema versions do not match",
			Details:     mismatches,
			Remediation: "Upgrade or reconcile members whose schema version differs from the active database schema.",
		})
	}
	if len(unavailable) > 0 {
		results = append(results, types.InspectionResult{
			ID:              "database-schema-coverage-" + database,
			Category:        "database",
			Status:          types.InspectionStatusUnknown,
			Summary:         strings.ToUpper(database) + " schema evidence is incomplete",
			Details:         unavailable,
			Remediation:     "Restore schema collection or upgrade members with unsupported schema APIs.",
			CollectionError: strings.Join(collectionErrors, "; "),
		})
	}
	if len(results) > 0 {
		return results
	}

	return []types.InspectionResult{{
		ID:       "database-schema-" + database,
		Category: "database",
		Status:   types.InspectionStatusPass,
		Summary:  strings.ToUpper(database) + " schema versions match",
	}}
}

func schemaCoverageDetail(node string, summary string) types.InspectionDetail {
	return types.InspectionDetail{
		Node:    node,
		ID:      "schema-version",
		Status:  types.InspectionStatusUnknown,
		Summary: summary,
	}
}

func databaseCommunicationResult(evidence types.InspectionCommunicationEvidence) types.InspectionResult {
	summary := summarizeCommunication(evidence)
	result := types.InspectionResult{
		ID:              "database-communication",
		Category:        "database",
		Status:          summary.Status,
		Summary:         summary.Message,
		CollectionError: evidence.CollectionError,
		Details: []types.InspectionDetail{{
			ID:      "database-communication",
			Status:  summary.Status,
			Summary: summary.Message,
			Data:    communicationData(evidence),
		}},
	}

	switch summary.Status {
	case types.InspectionStatusPass:
		result.Summary = "Database communication is healthy"
		result.Details[0].Summary = result.Summary
	case types.InspectionStatusWarning:
		result.Remediation = "Investigate Southbound database processing and convergence."
	case types.InspectionStatusFail:
		result.Remediation = "Restore connectivity to the unreachable OVN database."
	case types.InspectionStatusUnknown:
		result.Remediation = "Restore database communication evidence collection."
	}

	return result
}

func communicationData(evidence types.InspectionCommunicationEvidence) map[string]string {
	data := make(map[string]string)
	if evidence.NBCfg != nil {
		data["nb_cfg"] = fmt.Sprintf("%d", *evidence.NBCfg)
	}
	if evidence.SBCfg != nil {
		data["sb_cfg"] = fmt.Sprintf("%d", *evidence.SBCfg)
	}
	if evidence.NBReachable != nil {
		data["nb_reachable"] = fmt.Sprintf("%t", *evidence.NBReachable)
	}
	if evidence.SBReachable != nil {
		data["sb_reachable"] = fmt.Sprintf("%t", *evidence.SBReachable)
	}
	if evidence.Converged != nil {
		data["converged"] = fmt.Sprintf("%t", *evidence.Converged)
	}
	return data
}

func nonAuthoritativeDatabaseResult(input Input) types.InspectionResult {
	result := types.InspectionResult{
		ID:          "database",
		Category:    "database",
		Status:      types.InspectionStatusUnknown,
		Summary:     "Cluster database health cannot be determined from a non-voting member",
		Remediation: "Run the inspection on a voting member.",
	}

	for _, database := range []string{"nb", "sb"} {
		evidence := input.Schemas[database]
		if evidence.ActiveVersion == "" {
			continue
		}
		result.Details = append(result.Details, types.InspectionDetail{
			ID:      "local-schema-" + database,
			Status:  types.InspectionStatusUnknown,
			Summary: "Local " + strings.ToUpper(database) + " active schema was collected",
			Data:    map[string]string{"active_version": evidence.ActiveVersion},
		})
	}

	if data := communicationData(input.Communication); len(data) > 0 {
		result.Details = append(result.Details, types.InspectionDetail{
			ID:      "local-communication",
			Status:  types.InspectionStatusUnknown,
			Summary: "Local database communication evidence was collected",
			Data:    data,
		})
	}

	var collectionErrors []string
	if input.Communication.CollectionError != "" {
		collectionErrors = append(collectionErrors, input.Communication.CollectionError)
	}
	if collectionError := databaseCollectionError(input); collectionError != "" {
		collectionErrors = append(collectionErrors, collectionError)
	}
	slices.Sort(collectionErrors)
	result.CollectionError = strings.Join(collectionErrors, "; ")
	return result
}

func databaseCollectionError(input Input) string {
	var errors []string
	for _, collectionError := range input.CollectionErrors {
		if collectionError.FactGroup == "database" {
			errors = append(errors, collectionError.Message)
		}
	}
	slices.Sort(errors)
	return strings.Join(errors, "; ")
}
