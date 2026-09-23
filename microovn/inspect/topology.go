package inspect

import (
	"context"

	"github.com/canonical/microovn/microovn/api/types"
)

type TopologyCheck struct{}

func (TopologyCheck) Name() string {
	return "topology"
}

func (TopologyCheck) Run(_ context.Context, input Input) []types.InspectionResult {
	if !input.ExecutionContext.Authoritative {
		return []types.InspectionResult{nonAuthoritativeTopologyResult()}
	}
	if !input.DesiredStateAvailable {
		return []types.InspectionResult{desiredStateUnavailableTopologyResult(input)}
	}

	centralCnt := 0
	for _, service := range input.Services {
		if service.Service == types.SrvCentral {
			centralCnt++
		}
	}

	if centralCnt == 0 {
		return []types.InspectionResult{{
			ID:       "central-topology",
			Category: "cluster",
			Status:   types.InspectionStatusPass,
			Summary:  "Central topology is not managed by this deployment",
		}}
	}

	var results []types.InspectionResult
	if centralCnt%2 == 0 {
		results = append(results, types.InspectionResult{
			ID:          "central-count-even",
			Category:    "cluster",
			Status:      types.InspectionStatusWarning,
			Summary:     "Central topology has an even number of central services",
			Remediation: "Configure an odd number of central services, with at least three for fault tolerance.",
		})
	}
	if centralCnt < 3 {
		results = append(results, types.InspectionResult{
			ID:          "central-count-few",
			Category:    "cluster",
			Status:      types.InspectionStatusWarning,
			Summary:     "Central topology has fewer than three central services",
			Remediation: "Configure at least three central services to tolerate a node failure.",
		})
	}
	if len(results) == 0 {
		results = append(results, types.InspectionResult{
			ID:       "central-topology",
			Category: "cluster",
			Status:   types.InspectionStatusPass,
			Summary:  "Central topology is healthy",
		})
	}
	return results
}

func nonAuthoritativeTopologyResult() types.InspectionResult {
	return types.InspectionResult{
		ID:          "central-topology",
		Category:    "cluster",
		Status:      types.InspectionStatusUnknown,
		Summary:     "Central topology is unavailable from a non-voting member",
		Remediation: "Run the inspection on a voting member.",
	}
}

func desiredStateUnavailableTopologyResult(input Input) types.InspectionResult {
	result := types.InspectionResult{
		ID:       "central-topology",
		Category: "cluster",
		Status:   types.InspectionStatusUnknown,
		Summary:  "Central topology could not be determined",
	}

	for _, collectionErr := range input.CollectionErrors {
		if collectionErr.FactGroup == "services" {
			result.CollectionError = collectionErr.Message
			break
		}
	}
	if result.CollectionError == "" {
		for _, collectionErr := range input.CollectionErrors {
			if collectionErr.FactGroup == "member" {
				result.CollectionError = collectionErr.Message
				break
			}
		}
	}

	return result
}
