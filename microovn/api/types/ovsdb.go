package types

// OvsdbSchemaFetchError is a collection of error types that can be returned when fetching
// OVSDB schema version via MicroOVN API.
type OvsdbSchemaFetchError int

const (
	OvsdbSchemaFetchErrorNone         OvsdbSchemaFetchError = iota // No error occurred
	OvsdbSchemaFetchErrorGeneric                                   // General catch-all error that did not fit more specific definition
	OvsdbSchemaFetchErrorNotSupported                              // API endpoint returned 404, signaling that the node does not support it.
)

// OvsdbSchemaReport is just a collection of OvsdbSchemaVersionResult structs
type OvsdbSchemaReport = []OvsdbSchemaVersionResult

// OvsdbSchemaVersionResult is a rich representation of a schema result fetch. It encapsulates node's Hostname,
// OVSDB schema version, and whether there were any error while fetching data from this node.
type OvsdbSchemaVersionResult struct {
	Host          string                `json:"host"`
	SchemaVersion string                `json:"schemaVersion"`
	Error         OvsdbSchemaFetchError `json:"error"`
}
