package inspect

import (
	"slices"
	"strings"

	"github.com/canonical/microovn/microovn/api/types"
)

type memberSnapshotError struct {
	node          string
	snapshotFound bool
	err           string
}

func forEachMemberSnapshot(
	input Input,
	factGroup string,
	callback func(string, types.InspectionNodeSnapshot),
) []memberSnapshotError {
	var unavailable []memberSnapshotError
	for _, node := range getSortedMemberNames(input) {
		snapshot, found := input.Snapshots[node]
		if !found {
			unavailable = append(unavailable, memberSnapshotError{
				node: node,
				err:  snapshotCollectionError(input, node),
			})
			continue
		}

		var errors []string
		for _, collectionError := range snapshot.Errors {
			if collectionError.FactGroup == factGroup {
				errors = append(errors, collectionError.Message)
			}
		}
		if len(errors) > 0 {
			slices.Sort(errors)
			unavailable = append(unavailable, memberSnapshotError{
				node:          node,
				snapshotFound: true,
				err:           strings.Join(errors, "; "),
			})
			continue
		}

		callback(node, snapshot)
	}

	return unavailable
}

func snapshotCollectionError(input Input, node string) string {
	var errors []string
	for _, collectionError := range input.CollectionErrors {
		if collectionError.Node == node &&
			(collectionError.FactGroup == "member" || collectionError.FactGroup == "snapshot") {
			errors = append(errors, collectionError.Message)
		}
	}
	slices.Sort(errors)
	return strings.Join(errors, "; ")
}

func getSortedMemberNames(input Input) []string {
	memberNames := make([]string, 0, len(input.Members))
	for _, member := range input.Members {
		memberNames = append(memberNames, member.Name)
	}
	slices.Sort(memberNames)
	return memberNames
}
