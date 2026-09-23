package inspect

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/canonical/microovn/microovn/api/types"
)

type EnvironmentCheck struct{}

var memberSpecificEnvironment = map[string]struct{}{
	"OVN_INITIAL_NB": {},
	"OVN_INITIAL_SB": {},
	"OVN_LOCAL_IP":   {},
}

func (EnvironmentCheck) Name() string {
	return "environment"
}

func (EnvironmentCheck) Run(_ context.Context, input Input) []types.InspectionResult {
	if !input.ExecutionContext.Authoritative {
		return []types.InspectionResult{nonAuthoritativeEnvironmentResult(input)}
	}

	var complete []environmentSnapshot
	var unavailable []types.InspectionDetail
	var collectionErrors []string
	memberErrors := forEachMemberSnapshot(input, "environment", func(memberName string, snapshot types.InspectionNodeSnapshot) {
		complete = append(complete, environmentSnapshot{
			node:        memberName,
			environment: canonicalEnvironment(snapshot.Environment),
		})
	})
	for _, memberError := range memberErrors {
		summary := "Environment snapshot is unavailable"
		if memberError.snapshotFound {
			summary = "Environment collection failed"
		}
		unavailable = append(unavailable, types.InspectionDetail{
			Node:    memberError.node,
			ID:      "environment-unavailable",
			Status:  types.InspectionStatusUnknown,
			Summary: summary,
		})
		if memberError.err != "" {
			collectionErrors = append(collectionErrors, fmt.Sprintf("%s: %s", memberError.node, memberError.err))
		}
	}

	if len(input.Members) == 0 {
		unavailable = append(unavailable, types.InspectionDetail{
			ID:      "environment-unavailable",
			Status:  types.InspectionStatusUnknown,
			Summary: "Expected cluster members are unavailable",
		})
	}

	groups := groupEnvironments(complete)
	var results []types.InspectionResult
	if len(groups) > 1 {
		results = append(results, environmentDriftResult(groups))
	}

	if len(unavailable) > 0 {
		results = append(results, types.InspectionResult{
			ID:              "environment-coverage",
			Category:        "environment",
			Status:          types.InspectionStatusUnknown,
			Summary:         "Environment evidence is incomplete",
			Details:         unavailable,
			Remediation:     "Restore environment snapshot collection on unavailable members.",
			CollectionError: strings.Join(collectionErrors, "; "),
		})
	}

	if len(results) > 0 {
		return results
	}

	return []types.InspectionResult{{
		ID:       "environment",
		Category: "environment",
		Status:   types.InspectionStatusPass,
		Summary:  "Member environments are consistent",
		Details:  environmentGroupDetails(groups),
	}}
}

func nonAuthoritativeEnvironmentResult(input Input) types.InspectionResult {
	result := types.InspectionResult{
		ID:          "environment",
		Category:    "environment",
		Status:      types.InspectionStatusUnknown,
		Summary:     "Environment convergence cannot be determined from a non-voting member",
		Remediation: "Run the inspection on a voting member.",
	}
	if snapshot, found := input.Snapshots[input.ExecutionContext.LocalNode]; found {
		if collectionError := environmentCollectionError(snapshot); collectionError != "" {
			result.CollectionError = collectionError
		} else {
			result.Details = []types.InspectionDetail{{
				Node:    snapshot.NodeName,
				ID:      "local-fingerprint",
				Status:  types.InspectionStatusUnknown,
				Summary: "Local environment fingerprint was collected",
				Data:    environmentData(canonicalEnvironment(snapshot.Environment)),
			}}
		}
	} else {
		result.CollectionError = snapshotCollectionError(input, input.ExecutionContext.LocalNode)
	}

	return result
}

type environmentSnapshot struct {
	node        string
	environment []types.InspectionEnvironment
}

type environmentGroup struct {
	nodes       []string
	environment []types.InspectionEnvironment
}

type diffResult struct {
	added          []types.InspectionEnvironment
	removed        []types.InspectionEnvironment
	hashMismatches []hashMismatch
}

type hashMismatch struct {
	name    string
	hashInA string
	hashInB string
}

func environmentCollectionError(snapshot types.InspectionNodeSnapshot) string {
	index := slices.IndexFunc(snapshot.Errors, func(collectionError types.InspectionCollectionError) bool {
		return collectionError.FactGroup == "environment"
	})
	if index == -1 {
		return ""
	}

	return snapshot.Errors[index].Message
}

func canonicalEnvironment(environment []types.InspectionEnvironment) []types.InspectionEnvironment {
	canonical := make([]types.InspectionEnvironment, 0, len(environment))
	for _, entry := range environment {
		if _, memberSpecific := memberSpecificEnvironment[entry.Name]; memberSpecific {
			continue
		}
		canonical = append(canonical, entry)
	}
	slices.SortFunc(canonical, func(a, b types.InspectionEnvironment) int {
		if result := cmp.Compare(a.Name, b.Name); result != 0 {
			return result
		}
		return cmp.Compare(a.Hash, b.Hash)
	})
	return canonical
}

func environmentFingerprint(environment []types.InspectionEnvironment) string {
	var fingerprint strings.Builder
	for _, entry := range environment {
		fmt.Fprintf(&fingerprint, "%d:%s%d:%s", len(entry.Name), entry.Name, len(entry.Hash), entry.Hash)
	}
	return fingerprint.String()
}

func groupEnvironments(snapshots []environmentSnapshot) []environmentGroup {
	groups := make([]environmentGroup, 0)
	groupByFingerprint := make(map[string]int)
	for _, snapshot := range snapshots {
		fingerprint := environmentFingerprint(snapshot.environment)
		index, found := groupByFingerprint[fingerprint]
		if !found {
			index = len(groups)
			groupByFingerprint[fingerprint] = index
			groups = append(groups, environmentGroup{environment: snapshot.environment})
		}
		groups[index].nodes = append(groups[index].nodes, snapshot.node)
	}

	return groups
}

func environmentData(environment []types.InspectionEnvironment) map[string]string {
	data := make(map[string]string, len(environment))
	for _, entry := range environment {
		data[entry.Name] = entry.Hash
	}
	return data
}

func environmentGroupDetails(groups []environmentGroup) []types.InspectionDetail {
	details := make([]types.InspectionDetail, 0, len(groups))
	for index, group := range groups {
		details = append(details, types.InspectionDetail{
			Node:    group.nodes[0],
			ID:      fmt.Sprintf("fingerprint-%d", index+1),
			Status:  types.InspectionStatusPass,
			Summary: fmt.Sprintf("Environment fingerprint shared by %s", strings.Join(group.nodes, ", ")),
			Data:    environmentData(group.environment),
		})
	}
	return details
}

func environmentDriftResult(groups []environmentGroup) types.InspectionResult {
	details := environmentGroupDetails(groups[:1])
	details[0].Status = types.InspectionStatusWarning
	details[0].Summary = fmt.Sprintf("Baseline environment fingerprint shared by %s", strings.Join(groups[0].nodes, ", "))

	for index, group := range groups[1:] {
		data := make(map[string]string)
		diff := diffEnvironments(groups[0].environment, group.environment)
		for _, entry := range diff.added {
			data["added."+entry.Name] = entry.Hash
		}
		for _, entry := range diff.removed {
			data["removed."+entry.Name] = entry.Hash
		}
		for _, mismatch := range diff.hashMismatches {
			data["changed."+mismatch.name+".baseline"] = mismatch.hashInA
			data["changed."+mismatch.name+".observed"] = mismatch.hashInB
		}
		details = append(details, types.InspectionDetail{
			Node:    group.nodes[0],
			ID:      fmt.Sprintf("fingerprint-%d", index+2),
			Status:  types.InspectionStatusWarning,
			Summary: fmt.Sprintf("Environment fingerprint differs on %s", strings.Join(group.nodes, ", ")),
			Data:    data,
		})
	}

	return types.InspectionResult{
		ID:          "environment-drift",
		Category:    "environment",
		Status:      types.InspectionStatusWarning,
		Summary:     fmt.Sprintf("Member environments differ across %d fingerprints", len(groups)),
		Details:     details,
		Remediation: "Compare the generated environment configuration on members with different fingerprints.",
	}
}

func diffEnvironments(a, b []types.InspectionEnvironment) diffResult {
	var diff diffResult
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		l, r := a[i], b[j]

		switch {
		case l.Name == r.Name:
			if l.Hash != r.Hash {
				diff.hashMismatches = append(diff.hashMismatches, hashMismatch{
					name:    l.Name,
					hashInA: l.Hash,
					hashInB: r.Hash,
				})
			}
			i++
			j++
		case l.Name < r.Name:
			diff.removed = append(diff.removed, l)
			i++
		case l.Name > r.Name:
			diff.added = append(diff.added, r)
			j++
		}

	}

	for ; i < len(a); i++ {
		diff.removed = append(diff.removed, a[i])
	}

	for ; j < len(b); j++ {
		diff.added = append(diff.added, b[j])
	}

	return diff
}
