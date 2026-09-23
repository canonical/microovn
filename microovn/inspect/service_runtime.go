package inspect

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/canonical/microovn/microovn/api/types"
)

type ServiceRuntimeCheck struct{}

func (ServiceRuntimeCheck) Name() string {
	return "service-runtime"
}

func (ServiceRuntimeCheck) Run(_ context.Context, input Input) []types.InspectionResult {
	if !input.ExecutionContext.Authoritative {
		return []types.InspectionResult{nonAuthoritativeServiceRuntimeResult(input)}
	}

	if !input.DesiredStateAvailable {
		return []types.InspectionResult{desiredStateUnavailableServiceRuntimeResult(input)}
	}

	desired := buildDesiredServiceRuntime(input.Services)

	var complete []serviceRuntimeSnapshot
	var unavailable []types.InspectionDetail
	var collectionErrors []string
	memberErrors := forEachMemberSnapshot(input, "daemons", func(memberName string, snapshot types.InspectionNodeSnapshot) {
		complete = append(complete, serviceRuntimeSnapshot{
			node:     memberName,
			services: canonicalServices(snapshot.Daemons),
		})
	})
	for _, memberError := range memberErrors {
		summary := "Service runtime snapshot is unavailable"
		if memberError.snapshotFound {
			summary = "Service runtime collection failed"
		}
		unavailable = append(unavailable, types.InspectionDetail{
			Node:    memberError.node,
			ID:      "service-runtime-unavailable",
			Status:  types.InspectionStatusUnknown,
			Summary: summary,
		})
		if memberError.err != "" {
			collectionErrors = append(collectionErrors, fmt.Sprintf("%s: %s", memberError.node, memberError.err))
		}
	}

	if len(input.Members) == 0 {
		unavailable = append(unavailable, types.InspectionDetail{
			ID:      "service-runtime-unavailable",
			Status:  types.InspectionStatusUnknown,
			Summary: "Expected cluster members are unavailable",
		})
	}

	var failures []types.InspectionDetail
	var warnings []types.InspectionDetail
	for _, snapshot := range complete {
		snapshotFailures, snapshotWarnings := serviceRuntimeDetails(snapshot, desired[snapshot.node])
		failures = append(failures, snapshotFailures...)
		warnings = append(warnings, snapshotWarnings...)
	}

	var results []types.InspectionResult
	if len(failures) > 0 {
		results = append(results, types.InspectionResult{
			ID:          "service-runtime-failure",
			Category:    "service-runtime",
			Status:      types.InspectionStatusFail,
			Summary:     "Required services are not active and enabled",
			Details:     failures,
			Remediation: "Ensure that required services are active and enabled on the affected members.",
		})
	}
	if len(warnings) > 0 {
		results = append(results, types.InspectionResult{
			ID:          "service-runtime-drift",
			Category:    "service-runtime",
			Status:      types.InspectionStatusWarning,
			Summary:     "Unexpected services are active or enabled",
			Details:     warnings,
			Remediation: "Disable unexpected services after confirming they are no longer needed.",
		})
	}
	if len(unavailable) > 0 {
		results = append(results, types.InspectionResult{
			ID:              "service-runtime-coverage",
			Category:        "service-runtime",
			Status:          types.InspectionStatusUnknown,
			Summary:         "Service runtime evidence is incomplete",
			Details:         unavailable,
			Remediation:     "Restore service runtime snapshot collection on unavailable members.",
			CollectionError: strings.Join(collectionErrors, "; "),
		})
	}

	if len(results) > 0 {
		return results
	}

	return []types.InspectionResult{{
		ID:       "service-runtime",
		Category: "service-runtime",
		Status:   types.InspectionStatusPass,
		Summary:  "Service runtime is consistent across cluster members",
		Details:  serviceDetails(complete),
	}}
}

var roleDaemons = []struct {
	role    types.SrvName
	daemons []string
}{
	{role: types.SrvBgp, daemons: []string{"microovn.bird"}},
	{role: types.SrvChassis, daemons: []string{"microovn.chassis"}},
	{role: types.SrvCentral, daemons: []string{
		"microovn.ovn-northd",
		"microovn.ovn-ovsdb-server-nb",
		"microovn.ovn-ovsdb-server-sb",
	}},
	{role: types.SrvSwitch, daemons: []string{"microovn.switch"}},
}

func nonAuthoritativeServiceRuntimeResult(input Input) types.InspectionResult {
	result := types.InspectionResult{
		ID:          "service-runtime",
		Category:    "service-runtime",
		Status:      types.InspectionStatusUnknown,
		Summary:     "Service runtime convergence cannot be determined from a non-voting member",
		Remediation: "Run the inspection on a voting member.",
	}
	if snapshot, found := input.Snapshots[input.ExecutionContext.LocalNode]; found {
		if collectionError := daemonCollectionError(snapshot); collectionError != "" {
			result.CollectionError = collectionError
			return result
		}
		result.Details = []types.InspectionDetail{{
			Node:    snapshot.NodeName,
			ID:      "local-fingerprint",
			Status:  types.InspectionStatusUnknown,
			Summary: "Local service runtime fingerprint was collected",
			Data:    serviceRuntimeData(snapshot.Daemons),
		}}
	} else {
		result.CollectionError = snapshotCollectionError(input, input.ExecutionContext.LocalNode)
	}

	return result
}

func desiredStateUnavailableServiceRuntimeResult(input Input) types.InspectionResult {
	return types.InspectionResult{
		ID:              "service-runtime-coverage",
		Category:        "service-runtime",
		Status:          types.InspectionStatusUnknown,
		Summary:         "Desired service state could not be determined",
		Remediation:     "Restore desired service state collection and run the inspection again.",
		CollectionError: desiredServiceCollectionError(input),
	}
}

type serviceRuntimeSnapshot struct {
	node     string
	services []types.InspectionDaemonState
}

func serviceRuntimeData(daemons []types.InspectionDaemonState) map[string]string {
	data := make(map[string]string, len(daemons))
	for _, daemon := range daemons {
		data[daemon.Name] = fmt.Sprintf("active=%t,enabled=%t", daemon.Active, daemon.Enabled)
	}

	return data
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func canonicalServices(services []types.InspectionDaemonState) []types.InspectionDaemonState {
	canonical := slices.Clone(services)
	slices.SortFunc(canonical, func(a, b types.InspectionDaemonState) int {
		if result := cmp.Compare(a.Name, b.Name); result != 0 {
			return result
		}
		if result := cmp.Compare(boolToInt(a.Active), boolToInt(b.Active)); result != 0 {
			return result
		}
		return cmp.Compare(boolToInt(a.Enabled), boolToInt(b.Enabled))
	})
	return canonical
}

func daemonCollectionError(snapshot types.InspectionNodeSnapshot) string {
	var errors []string
	for _, collectionError := range snapshot.Errors {
		if collectionError.FactGroup == "daemons" {
			errors = append(errors, collectionError.Message)
		}
	}
	slices.Sort(errors)
	return strings.Join(errors, "; ")
}

func desiredServiceCollectionError(input Input) string {
	var errors []string
	for _, collectionError := range input.CollectionErrors {
		if collectionError.FactGroup == "services" || collectionError.FactGroup == "member" {
			errors = append(errors, collectionError.Message)
		}
	}
	slices.Sort(errors)
	return strings.Join(errors, "; ")
}

func buildDesiredServiceRuntime(services types.Services) map[string]map[types.SrvName]bool {
	desired := make(map[string]map[types.SrvName]bool)

	for _, service := range services {
		if desired[service.Location] == nil {
			desired[service.Location] = make(map[types.SrvName]bool)
		}

		desired[service.Location][service.Service] = true
	}

	return desired
}

func serviceDetails(snapshots []serviceRuntimeSnapshot) []types.InspectionDetail {
	details := make([]types.InspectionDetail, 0, len(snapshots))
	for _, snapshot := range snapshots {
		details = append(details, types.InspectionDetail{
			Node:    snapshot.node,
			ID:      "service-runtime",
			Status:  types.InspectionStatusPass,
			Summary: "Service runtime evidence was collected",
			Data:    serviceRuntimeData(snapshot.services),
		})
	}
	return details
}

func serviceRuntimeDetails(snapshot serviceRuntimeSnapshot, desired map[types.SrvName]bool) ([]types.InspectionDetail, []types.InspectionDetail) {
	observed := make(map[string]types.InspectionDaemonState, len(snapshot.services))
	for _, daemon := range snapshot.services {
		observed[daemon.Name] = daemon
	}

	var failures []types.InspectionDetail
	var warnings []types.InspectionDetail
	for _, entry := range roleDaemons {
		for _, daemonName := range entry.daemons {
			daemon, found := observed[daemonName]
			if desired[entry.role] {
				if !found || !daemon.Active || !daemon.Enabled {
					failures = append(failures, types.InspectionDetail{
						Node:    snapshot.node,
						ID:      daemonName,
						Status:  types.InspectionStatusFail,
						Summary: fmt.Sprintf("Required service %q is not active and enabled", daemonName),
						Data:    daemonStateData(daemon, found),
					})
				}
				continue
			}

			if found && (daemon.Active || daemon.Enabled) {
				warnings = append(warnings, types.InspectionDetail{
					Node:    snapshot.node,
					ID:      daemonName,
					Status:  types.InspectionStatusWarning,
					Summary: fmt.Sprintf("Unexpected service %q is active or enabled", daemonName),
					Data:    daemonStateData(daemon, true),
				})
			}
		}
	}

	return failures, warnings
}

func daemonStateData(daemon types.InspectionDaemonState, observed bool) map[string]string {
	return map[string]string{
		"observed": fmt.Sprintf("%t", observed),
		"active":   fmt.Sprintf("%t", daemon.Active),
		"enabled":  fmt.Sprintf("%t", daemon.Enabled),
	}
}
