package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/canonical/microcluster/v2/microcluster"
	"github.com/canonical/microovn/microovn/api/types"
	ovnCmd "github.com/canonical/microovn/microovn/ovn/cmd"
	"github.com/spf13/cobra"

	microClusterClient "github.com/canonical/microcluster/v2/client"
	"github.com/canonical/microovn/microovn/client"
)

type cmdStatus struct {
	common *CmdControl
}

func (c *cmdStatus) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Checks the cluster status",
		RunE:  c.Run,
	}

	return cmd
}

func (c *cmdStatus) Run(_ *cobra.Command, _ []string) error {
	m, err := microcluster.App(microcluster.Args{StateDir: c.common.FlagStateDir})
	if err != nil {
		return err
	}

	cli, err := m.LocalClient()
	if err != nil {
		return err
	}

	// Get services.
	services, err := client.GetServices(context.Background(), cli)
	if err != nil {
		return err
	}

	// Get cluster members.
	clusterMembers, err := cli.GetClusterMembers(context.Background())
	if err != nil {
		return err
	}

	fmt.Println("MicroOVN deployment summary:")

	for _, server := range clusterMembers {
		// Services.
		srvServices := []string{}
		for _, service := range services {
			if service.Location != server.Name {
				continue
			}

			srvServices = append(srvServices, service.Service)
		}
		sort.Strings(srvServices)

		fmt.Printf("- %s (%s)\n", server.Name, server.Address.Addr().String())
		fmt.Printf("  Services: %s\n", strings.Join(srvServices, ", "))
	}

	// Get OVN clustered DB schema version status
	fmt.Println("OVN Database summary:")
	reportOvsdbSchemaStatus(cli, ovnCmd.OvsdbTypeNBLocal)
	reportOvsdbSchemaStatus(cli, ovnCmd.OvsdbTypeSBLocal)
	return nil
}

// reportOvsdbSchemaStatus fetches currently active schema version and list of expected schema version from each
// node in the deployment. Based on the results it then prints a report for the user.
func reportOvsdbSchemaStatus(cli *microClusterClient.Client, ovsdbType ovnCmd.OvsdbType) {
	ovnDB, err := ovnCmd.NewOvsdbSpec(ovsdbType)
	if err != nil {
		printOvsdbSummaryError(err, nil)
		return
	}

	activeSchema, errType := client.GetActiveOvsdbSchemaVersion(context.Background(), cli, ovnDB)
	if errType != types.OvsdbSchemaFetchErrorNone {
		printOvsdbSummaryError(
			fmt.Errorf("failed to get OVN %s active schema version", ovnDB.FriendlyName),
			ovnDB,
		)
	}

	expectedSchemas, err := client.GetAllExpectedOvsdbSchemaVersions(context.Background(), cli, ovnDB)
	if err != nil {
		printOvsdbSummaryError(
			fmt.Errorf("failed to get expected OVN %s schema versions", ovnDB.FriendlyName),
			ovnDB,
		)
	}

	printOvsdbSchemaReport(cli, ovnDB, activeSchema, expectedSchemas)
}

// printOvsdbSchemaReport evaluates active and expected schema versions of given OVN database and prints the report. If
// there's no attention of a user required, it prints simple "OK" message, otherwise it prints detailed reported about
// the database's active schema version and schema versions expected on each node in the deployment.
func printOvsdbSchemaReport(cli *microClusterClient.Client, dbSpec *ovnCmd.OvsdbSpec, activeSchema string, expectedSchemas types.OvsdbSchemaReport) {
	attentionRequired, err := ovsdbSchemaRequiresAttention(activeSchema, expectedSchemas)
	if err != nil {
		printOvsdbSummaryError(err, dbSpec)
		return
	}

	if !attentionRequired {
		fmt.Printf("OVN %s: OK (%s)\n", dbSpec.FriendlyName, activeSchema)
		return
	}

	clusterMembers, _ := cli.GetClusterMembers(context.Background())

	addrToName := func(addr string) (string, error) {
		for _, member := range clusterMembers {
			if member.Address.Addr().String() == addr {
				return member.Name, nil
			}
		}
		return "", fmt.Errorf("cluster member with address '%s' not found", addr)
	}

	msg := fmt.Sprintf("OVN %s: Upgrade or attention required!\n", dbSpec.FriendlyName)
	msg += fmt.Sprintf("Currently active schema: %s\n", activeSchema)
	msg += "Cluster report (expected schema versions):\n"
	for _, node := range expectedSchemas {
		nodeName, err := addrToName(node.Host)
		if err != nil {
			nodeName = node.Host
		}
		msg += fmt.Sprintf("\t%s: ", nodeName)
		switch node.Error {
		case types.OvsdbSchemaFetchErrorGeneric:
			msg += "Error. Failed to contact member\n"
		case types.OvsdbSchemaFetchErrorNotSupported:
			msg += "Missing API. MicroOVN needs upgrade\n"
		default:
			msg += fmt.Sprintf("%s\n", node.SchemaVersion)
		}
	}
	msg += "\n"

	fmt.Print(msg)
}

// ovsdbSchemaRequiresAttention is a function that determines whether an attention of the user is needed for given OVN
// database. It takes currently active schema version, list of expected version and returns false if everything
// is as expected.
// It returns true if active version does not match expected version on all nodes or if there was any error reported
// in expectedSchemas.
// It returns error if expectedSchemas is an empty slice.
func ovsdbSchemaRequiresAttention(activeSchema string, expectedSchemas types.OvsdbSchemaReport) (bool, error) {
	if len(expectedSchemas) == 0 {
		return false, errors.New("list of expected schema versions is empty")
	}

	versionSet := make(map[string]int)
	nodeError := false
	expectedVersion := "0.0"

	// Iterate over all nodes and find unique expected versions as well as any errors
	for _, node := range expectedSchemas {
		expectedVersion = node.SchemaVersion
		versionSet[node.SchemaVersion]++
		if node.Error != types.OvsdbSchemaFetchErrorNone {
			nodeError = true
		}
	}

	// return True if any errors were encountered
	if nodeError {
		return true, nil
	}

	// return True if there's more than one unique schema version expected
	if len(versionSet) > 1 {
		return true, nil
	}

	// return True if expected schema version does not match the currently running version
	return activeSchema != expectedVersion, nil
}

// printOvsdbSummaryError prepends unified prefix, and prints the error. It should be used when aborting OVSDB
// summary report.
// if ovsdbSpec is provided, name od the database will be included in the printer error.
func printOvsdbSummaryError(err error, ovsdbSpec *ovnCmd.OvsdbSpec) {
	dbName := ""
	if ovsdbSpec != nil {
		dbName = ovsdbSpec.FriendlyName
	}
	fmt.Printf("Error creating OVN %s Database summary: %s\n", dbName, err)
}
