package inspect

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/canonical/microcluster/v3/microcluster"
	microTypes "github.com/canonical/microcluster/v3/microcluster/types"
	"github.com/canonical/microovn/microovn/api/types"
	"github.com/canonical/microovn/microovn/client"
	"github.com/canonical/microovn/microovn/ovn/cmd"
)

func collectInput(ctx context.Context, cluster *microcluster.MicroCluster, local microTypes.Client) (Input, error) {
	input := Input{
		Snapshots: make(map[string]types.InspectionNodeSnapshot),
		Schemas:   make(map[string]types.InspectionSchemaEvidence),
	}

	execution, err := collectExecutionContext(ctx, cluster)
	if err != nil {
		return Input{}, err
	}
	input.ExecutionContext = execution

	snapshot, err := client.GetNodeSnapshot(ctx, local)
	if err != nil {
		input.CollectionErrors = append(input.CollectionErrors, types.InspectionCollectionError{
			Node:      input.ExecutionContext.LocalNode,
			FactGroup: "snapshot",
			Message:   fmt.Sprintf("failed to get local node snapshot: %v", err),
		})
	} else {
		snapshot.NodeName = input.ExecutionContext.LocalNode
		input.Snapshots[snapshot.NodeName] = snapshot
	}

	if input.ExecutionContext.Authoritative {
		members, err := cluster.GetClusterMembers(ctx)
		if err != nil {
			input.CollectionErrors = append(input.CollectionErrors, types.InspectionCollectionError{
				Node:      input.ExecutionContext.LocalNode,
				FactGroup: "member",
				Message:   fmt.Sprintf("failed to get cluster members: %v", err),
			})
			return input, nil
		}
		slices.SortFunc(
			members,
			func(a, b microTypes.ClusterMember) int {
				return cmp.Compare(a.Name, b.Name)
			},
		)
		input.Members = members
		collectClusterEvidence(ctx, cluster, local, &input)
	} else {
		collectLocalEvidence(ctx, local, &input)
	}

	slices.SortFunc(
		input.CollectionErrors,
		func(a, b types.InspectionCollectionError) int {
			if result := cmp.Compare(a.Node, b.Node); result != 0 {
				return result
			}
			if result := cmp.Compare(a.FactGroup, b.FactGroup); result != 0 {
				return result
			}
			return cmp.Compare(a.Message, b.Message)
		},
	)

	return input, nil
}

func collectExecutionContext(ctx context.Context, cluster *microcluster.MicroCluster) (types.InspectionExecutionContext, error) {
	status, err := cluster.Status(ctx)
	if err != nil {
		return types.InspectionExecutionContext{}, fmt.Errorf("failed to get local member status: %w", err)
	}

	members, err := cluster.GetDqliteClusterMembers()
	if err != nil {
		return types.InspectionExecutionContext{}, dqliteMembersError(err)
	}

	return resolveExecutionContext(status, members)
}

func dqliteMembersError(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return errors.New("MicroOVN is not initialized")
	}

	return fmt.Errorf("failed to get local dqlite members: %w", err)
}

func resolveExecutionContext(status *microTypes.Server, members []microTypes.DqliteMember) (types.InspectionExecutionContext, error) {

	var matches []microTypes.DqliteMember
	for _, member := range members {
		if member.Name == status.Name {
			matches = append(matches, member)
		}
	}

	if len(matches) == 0 {
		for _, member := range members {
			if sameMemberAddress(member.Address, status.Address) {
				matches = append(matches, member)
			}
		}
	}

	if len(matches) != 1 {
		return types.InspectionExecutionContext{}, fmt.Errorf(
			"expected one local dqlite member, found %d",
			len(matches),
		)
	}

	role := matches[0].Role
	name := matches[0].Name
	execution := types.InspectionExecutionContext{
		LocalNode: name,
	}

	switch role {
	case string(types.InspectionMemberRoleVoter):
		execution.Authoritative = true
		execution.Scope = types.InspectionScopeCluster
		execution.MemberRole = types.InspectionMemberRoleVoter
	case string(types.InspectionMemberRoleSpare):
		execution.Scope = types.InspectionScopeLocal
		execution.MemberRole = types.InspectionMemberRoleSpare
	case "stand-by", string(types.InspectionMemberRoleStandby):
		// Normalize dqlite role to a canonical member role used in the inspection context
		execution.Scope = types.InspectionScopeLocal
		execution.MemberRole = types.InspectionMemberRoleStandby
	default:
		return types.InspectionExecutionContext{}, fmt.Errorf("unsupported local member role %q", role)
	}

	return execution, nil
}

func sameMemberAddress(memberAddress string, statusAddress microTypes.AddrPort) bool {
	if memberAddress == statusAddress.String() {
		return true
	}

	parsed, err := netip.ParseAddrPort(memberAddress)
	return err == nil && parsed.Addr() == statusAddress.Addr()
}

func collectLocalEvidence(ctx context.Context, local microTypes.Client, input *Input) {
	probe, err := client.GetDatabaseProbe(ctx, local, types.InspectionScopeLocal)
	if err != nil {
		input.CollectionErrors = append(input.CollectionErrors, types.InspectionCollectionError{
			Node:      input.ExecutionContext.LocalNode,
			FactGroup: "database",
			Message:   fmt.Sprintf("failed to get local database probe: %v", err),
		})
	}
	input.Communication = probe.Communication
	for _, schema := range probe.Schemas {
		input.Schemas[schema.Database] = schema
	}

	input.DesiredStateAvailable = false
}

func collectClusterEvidence(ctx context.Context, cluster *microcluster.MicroCluster, local microTypes.Client, input *Input) {
	input.DesiredStateAvailable = true

	// Services
	services, err := client.GetServices(ctx, local)
	if err != nil {
		input.CollectionErrors = append(input.CollectionErrors, types.InspectionCollectionError{
			Node:      input.ExecutionContext.LocalNode,
			FactGroup: "services",
			Message:   fmt.Sprintf("failed to get services: %v", err),
		})
		input.DesiredStateAvailable = false
	}
	input.Services = services

	// Database probe
	dbProbe, err := client.GetDatabaseProbe(ctx, local, types.InspectionScopeCluster)
	if err != nil {
		input.CollectionErrors = append(input.CollectionErrors, types.InspectionCollectionError{
			Node:      input.ExecutionContext.LocalNode,
			FactGroup: "database",
			Message:   fmt.Sprintf("failed to get local database probe: %v", err),
		})
	}
	input.Communication = dbProbe.Communication
	for _, schema := range dbProbe.Schemas {
		input.Schemas[schema.Database] = schema
	}

	// Schema
	for _, dbType := range []cmd.OvsdbType{
		cmd.OvsdbTypeNBLocal,
		cmd.OvsdbTypeSBLocal,
	} {
		evidence, errs := collectClusterSchema(ctx, local, dbType, input.Members)
		if evidence.Database != "" {
			input.Schemas[evidence.Database] = evidence
		}
		input.CollectionErrors = append(input.CollectionErrors, errs...)
	}

	// Cluster network probe
	netProbe, err := client.GetNetworkProbe(ctx, local)
	if err != nil {
		input.CollectionErrors = append(input.CollectionErrors, types.InspectionCollectionError{
			Node:      input.ExecutionContext.LocalNode,
			FactGroup: "network",
			Message:   fmt.Sprintf("failed to get network probe: %v", err),
		})
	}
	input.Network = netProbe

	// Remote snapshots
	var memberClients []memberClient
	for _, member := range input.Members {
		if member.Name == input.ExecutionContext.LocalNode {
			continue
		}
		mClient, err := cluster.RemoteClient(member.Address.String())
		if err != nil {
			input.CollectionErrors = append(input.CollectionErrors, types.InspectionCollectionError{
				Node:      member.Name,
				FactGroup: "member",
				Message:   fmt.Sprintf("failed to get member client: %v", err),
			})
			continue
		}
		memberClients = append(memberClients, memberClient{member: member, client: mClient})
	}

	snapshots, errs := getMemberNodeSnapshots(ctx, memberClients)
	input.CollectionErrors = append(input.CollectionErrors, errs...)
	for _, snapshot := range snapshots {
		input.Snapshots[snapshot.NodeName] = snapshot
	}
}

func collectClusterSchema(
	ctx context.Context,
	local microTypes.Client,
	dbType cmd.OvsdbType,
	members []microTypes.ClusterMember,
) (types.InspectionSchemaEvidence, []types.InspectionCollectionError) {
	spec, err := cmd.NewOvsdbSpec(dbType)
	if err != nil {
		return types.InspectionSchemaEvidence{}, []types.InspectionCollectionError{{
			FactGroup: "database",
			Message:   err.Error(),
		}}
	}

	evidence := types.InspectionSchemaEvidence{
		Database: spec.ShortName,
		Members:  make([]types.InspectionSchemaMemberEvidence, len(members)),
	}

	memberByAddress := make(map[string]int, len(members))
	for index, member := range members {
		evidence.Members[index] = types.InspectionSchemaMemberEvidence{
			Node:  member.Name,
			Error: "schema response missing",
		}
		memberByAddress[member.Address.Addr().String()] = index
	}

	active, activeResult := client.GetActiveOvsdbSchemaVersion(ctx, local, spec)
	switch activeResult {
	case types.OvsdbSchemaFetchErrorNone:
		evidence.ActiveVersion = active
	case types.OvsdbSchemaFetchErrorNotSupported:
		evidence.ActiveError = "active schema API is unsupported"
	default:
		evidence.ActiveError = "active schema version could not be collected"
	}

	report, reportErr := client.GetAllExpectedOvsdbSchemaVersions(ctx, local, spec)
	if reportErr != nil {
		return evidence, []types.InspectionCollectionError{{
			FactGroup: "database",
			Message: fmt.Sprintf(
				"failed to get expected %s schema versions: %v",
				spec.FriendlyName,
				reportErr,
			),
		}}
	}

	var collectionErrors []types.InspectionCollectionError
	for _, result := range report {
		index, found := memberByAddress[result.Host]
		if !found {
			collectionErrors = append(collectionErrors, types.InspectionCollectionError{
				FactGroup: "database",
				Message:   fmt.Sprintf("schema response from unknown address %q", result.Host),
			})
			continue
		}

		memberEvidence := types.InspectionSchemaMemberEvidence{
			Node: members[index].Name,
		}
		switch result.Error {
		case types.OvsdbSchemaFetchErrorNone:
			memberEvidence.Version = result.SchemaVersion
		case types.OvsdbSchemaFetchErrorNotSupported:
			memberEvidence.Unsupported = true
		default:
			memberEvidence.Error = "schema version could not be collected"
		}
		evidence.Members[index] = memberEvidence
	}

	slices.SortFunc(evidence.Members, func(a, b types.InspectionSchemaMemberEvidence) int {
		return cmp.Compare(a.Node, b.Node)
	})

	return evidence, collectionErrors
}

func getMemberNodeSnapshots(ctx context.Context, members []memberClient) ([]types.InspectionNodeSnapshot, []types.InspectionCollectionError) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var snapshots []types.InspectionNodeSnapshot
	var errors []types.InspectionCollectionError

	appendError := func(err types.InspectionCollectionError) {
		mu.Lock()
		defer mu.Unlock()
		errors = append(errors, err)
	}

	appendSnapshot := func(snapshot types.InspectionNodeSnapshot) {
		mu.Lock()
		defer mu.Unlock()
		snapshots = append(snapshots, snapshot)
	}

	sem := make(chan struct{}, 4)

	for _, m := range members {
		wg.Add(1)
		go func(m memberClient) {
			defer wg.Done()
			if err := ctx.Err(); err != nil {
				appendError(types.InspectionCollectionError{
					Node:      m.member.Name,
					FactGroup: "member",
					Message:   fmt.Sprintf("failed to get member node snapshot: %v", err),
				})
				return
			}

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				errors = append(errors, types.InspectionCollectionError{
					Node:      m.member.Name,
					FactGroup: "member",
					Message:   fmt.Sprintf("failed to get member node snapshot: %v", ctx.Err()),
				})
				mu.Unlock()
				return
			}

			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			snapshot, err := client.GetNodeSnapshot(ctx, m.client)
			if err != nil {
				appendError(types.InspectionCollectionError{
					Node:      m.member.Name,
					FactGroup: "member",
					Message:   fmt.Sprintf("failed to get member node snapshot: %v", err),
				})
				return
			}
			snapshot.NodeName = m.member.Name
			appendSnapshot(snapshot)
		}(m)
	}

	wg.Wait()

	slices.SortFunc(
		snapshots,
		func(a, b types.InspectionNodeSnapshot) int {
			return cmp.Compare(a.NodeName, b.NodeName)
		},
	)
	slices.SortFunc(
		errors,
		func(a, b types.InspectionCollectionError) int {
			if result := cmp.Compare(a.Node, b.Node); result != 0 {
				return result
			}
			return cmp.Compare(a.FactGroup, b.FactGroup)
		},
	)

	return snapshots, errors
}
