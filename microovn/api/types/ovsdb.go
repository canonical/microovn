package types

// OvsdbSchemaFetchError is a collection of error types that can be returned when fetching
// OVSDB schema version via MicroOVN API.
type OvsdbSchemaFetchError int

const (
	OvsdbSchemaFetchErrorNone         OvsdbSchemaFetchError = iota // No error occurred
	OvsdbSchemaFetchErrorGeneric                                   // General catch-all error that did not fit more specific definition
	OvsdbSchemaFetchErrorNotSupported                              // API endpoint returned 404, signaling that the node does not support it.
)
