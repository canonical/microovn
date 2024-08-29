package main

import (
	"testing"

	"github.com/canonical/microovn/microovn/api/types"
)

func _ovsdbSchemaRequiresAttention(
	clusterSchema []string, nodeError []types.OvsdbSchemaFetchError,
	activeSchema string, expectedResult bool, t *testing.T,
) {
	expectedSchemas := types.OvsdbSchemaReport{
		types.OvsdbSchemaVersionResult{
			Host:          "a",
			SchemaVersion: clusterSchema[0],
			Error:         nodeError[0],
		},
		types.OvsdbSchemaVersionResult{
			Host:          "b",
			SchemaVersion: clusterSchema[1],
			Error:         nodeError[1],
		},
		types.OvsdbSchemaVersionResult{
			Host:          "c",
			SchemaVersion: clusterSchema[2],
			Error:         nodeError[2],
		},
	}
	result, _ := ovsdbSchemaRequiresAttention(activeSchema, expectedSchemas)
	if result != expectedResult {
		t.Fatalf("ovsdbSchemaRequiresAttention(%s, %q) returned %t, "+
			"expected %t.",
			activeSchema, expectedSchemas, result, expectedResult)
	}
}

func TestUnexported_ovsdbSchemaRequiresAttentionMatch(t *testing.T) {
	activeSchema := "1.0"
	clusterSchema := []string{activeSchema, activeSchema, activeSchema}
	nodeError := []types.OvsdbSchemaFetchError{0, 0, 0}
	_ovsdbSchemaRequiresAttention(clusterSchema, nodeError, activeSchema,
		false, t)
}

func TestUnexported_ovsdbSchemaRequiresAttentionMatchError(t *testing.T) {
	activeSchema := "1.0"
	clusterSchema := []string{activeSchema, activeSchema, activeSchema}
	nodeError := []types.OvsdbSchemaFetchError{0, 1, 0}
	_ovsdbSchemaRequiresAttention(clusterSchema, nodeError, activeSchema,
		true, t)
}

func TestUnexported_ovsdbSchemaRequiresAttentionMisMatch(t *testing.T) {
	activeSchema := "1.0"
	clusterSchema := []string{"1.1", "1.1", "1.1"}
	nodeError := []types.OvsdbSchemaFetchError{0, 0, 0}
	_ovsdbSchemaRequiresAttention(clusterSchema, nodeError, activeSchema,
		true, t)
}

func TestUnexported_ovsdbSchemaRequiresAttentionClusterDiff(t *testing.T) {
	activeSchema := "1.0"
	clusterSchema := []string{"1.1", "1.2", "1.0"}
	nodeError := []types.OvsdbSchemaFetchError{0, 0, 0}
	_ovsdbSchemaRequiresAttention(clusterSchema, nodeError, activeSchema,
		true, t)
}
