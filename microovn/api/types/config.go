package types

// SetConfigRequest defines the structure of a request to change a configuration option value
type SetConfigRequest struct {
	Key   string `json:"key"`   // Named of the configuration option
	Value string `json:"value"` // New value of the configuration option
}

// SetConfigResponse defines the structure of a response to a request for a configuration change.
type SetConfigResponse struct {
	Error string `json:"error"` // Description of an error that occurred. Empty on success.
}

// GetConfigRequest defines the structure of a request to get a value of a configuration option
type GetConfigRequest struct {
	Key string `json:"key"` // name of the configuration option
}

// GetConfigResponse fines the structure of a response to get current value fo a configuration option
type GetConfigResponse struct {
	Value string `json:"value"` // Current configuration option value. Empty on error or if the option is not set
	IsSet bool   `json:"isSet"` // Signals whether the config option is explicitly set.
	Error string `json:"error"` // Description of an error that occurred. Empty on success.
}

// DeleteConfigRequest defines the structure of a request to remove configuration option
type DeleteConfigRequest = GetConfigRequest

// DeleteConfigResponse defines the structure of a response to the request for removal of a configuration option
type DeleteConfigResponse = SetConfigResponse
