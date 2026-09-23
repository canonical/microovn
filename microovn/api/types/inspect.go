package types

import "time"

const (
	InspectionSchemaVersion = 1
)

type InspectionScope string

const (
	InspectionScopeCluster InspectionScope = "cluster"
	InspectionScopeLocal   InspectionScope = "local"
)

type InspectionMemberRole string

const (
	InspectionMemberRoleVoter   InspectionMemberRole = "voter"
	InspectionMemberRoleStandby InspectionMemberRole = "standby"
	InspectionMemberRoleSpare   InspectionMemberRole = "spare"
)

type InspectionReport struct {
	SchemaVersion    int                        `json:"schema_version" yaml:"schema_version"`
	Timestamp        time.Time                  `json:"timestamp" yaml:"timestamp"`
	ExecutionContext InspectionExecutionContext `json:"execution_context" yaml:"execution_context"`
	Summary          InspectionSummary          `json:"summary" yaml:"summary"`
	DatabaseSummary  InspectionDatabaseSummary  `json:"database_summary" yaml:"database_summary"`
	Results          []InspectionResult         `json:"results" yaml:"results"`
}

type InspectionStatus string

const (
	InspectionStatusPass    InspectionStatus = "PASS"
	InspectionStatusWarning InspectionStatus = "WARNING"
	InspectionStatusFail    InspectionStatus = "FAIL"
	InspectionStatusUnknown InspectionStatus = "UNKNOWN"
)

type InspectionExecutionContext struct {
	LocalNode     string               `json:"local_node" yaml:"local_node"`
	MemberRole    InspectionMemberRole `json:"member_role" yaml:"member_role"`
	Authoritative bool                 `json:"authoritative" yaml:"authoritative"`
	Scope         InspectionScope      `json:"scope" yaml:"scope"`
}

type InspectionSummary struct {
	Status InspectionStatus `json:"status" yaml:"status"`
	Counts InspectionCounts `json:"counts" yaml:"counts"`
}

type InspectionCounts struct {
	Pass    int `json:"pass" yaml:"pass"`
	Warning int `json:"warning" yaml:"warning"`
	Fail    int `json:"fail" yaml:"fail"`
	Unknown int `json:"unknown" yaml:"unknown"`
}

type InspectionDatabaseSummary struct {
	Northbound    InspectionDatabaseStatus       `json:"northbound" yaml:"northbound"`
	Southbound    InspectionDatabaseStatus       `json:"southbound" yaml:"southbound"`
	Communication InspectionCommunicationSummary `json:"communication" yaml:"communication"`
}

type InspectionDatabaseStatus struct {
	Status              InspectionStatus `json:"status" yaml:"status"`
	ActiveSchemaVersion string           `json:"active_schema_version,omitempty" yaml:"active_schema_version,omitempty"`
	ExpectedMembers     int              `json:"expected_members" yaml:"expected_members"`
	RespondingMembers   int              `json:"responding_members" yaml:"responding_members"`
	Message             string           `json:"message,omitempty" yaml:"message,omitempty"`
}

type InspectionSchemaEvidence struct {
	Database      string                           `json:"database" yaml:"database"`
	ActiveVersion string                           `json:"active_version,omitempty" yaml:"active_version,omitempty"`
	ActiveError   string                           `json:"active_error,omitempty" yaml:"active_error,omitempty"`
	Members       []InspectionSchemaMemberEvidence `json:"members" yaml:"members"`
}

type InspectionSchemaMemberEvidence struct {
	Node        string `json:"node" yaml:"node"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Unsupported bool   `json:"unsupported,omitempty" yaml:"unsupported,omitempty"`
	Error       string `json:"error,omitempty" yaml:"error,omitempty"`
}

type InspectionCommunicationSummary struct {
	Status  InspectionStatus `json:"status" yaml:"status"`
	NBCfg   *int64           `json:"nb_cfg,omitempty" yaml:"nb_cfg,omitempty"`
	SBCfg   *int64           `json:"sb_cfg,omitempty" yaml:"sb_cfg,omitempty"`
	Message string           `json:"message,omitempty" yaml:"message,omitempty"`
}

type InspectionCommunicationEvidence struct {
	NBCfg           *int64 `json:"nb_cfg,omitempty" yaml:"nb_cfg,omitempty"`
	SBCfg           *int64 `json:"sb_cfg,omitempty" yaml:"sb_cfg,omitempty"`
	NBReachable     *bool  `json:"nb_reachable,omitempty" yaml:"nb_reachable,omitempty"`
	SBReachable     *bool  `json:"sb_reachable,omitempty" yaml:"sb_reachable,omitempty"`
	Converged       *bool  `json:"converged,omitempty" yaml:"converged,omitempty"`
	CollectionError string `json:"collection_error,omitempty" yaml:"collection_error,omitempty"`
}

type InspectionResult struct {
	ID              string             `json:"id" yaml:"id"`
	Category        string             `json:"category" yaml:"category"`
	Status          InspectionStatus   `json:"status" yaml:"status"`
	Summary         string             `json:"summary" yaml:"summary"`
	Details         []InspectionDetail `json:"details,omitempty" yaml:"details,omitempty"`
	Remediation     string             `json:"remediation,omitempty" yaml:"remediation,omitempty"`
	CollectionError string             `json:"collection_error,omitempty" yaml:"collection_error,omitempty"`
}

type InspectionDetail struct {
	Node    string            `json:"node,omitempty" yaml:"node,omitempty"`
	ID      string            `json:"id" yaml:"id"`
	Status  InspectionStatus  `json:"status,omitempty" yaml:"status,omitempty"`
	Summary string            `json:"summary" yaml:"summary"`
	Data    map[string]string `json:"data,omitempty" yaml:"data,omitempty"`
}

type InspectionNodeSnapshot struct {
	NodeName    string                      `json:"node_name" yaml:"node_name"`
	Address     string                      `json:"address" yaml:"address"`
	Daemons     []InspectionDaemonState     `json:"daemons" yaml:"daemons"`
	Environment []InspectionEnvironment     `json:"environment" yaml:"environment"`
	Errors      []InspectionCollectionError `json:"errors,omitempty" yaml:"errors,omitempty"`
}

type InspectionDaemonState struct {
	Name    string `json:"name" yaml:"name"`
	Active  bool   `json:"active" yaml:"active"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
}

type InspectionEnvironment struct {
	Name string `json:"name" yaml:"name"`
	Hash string `json:"hash" yaml:"hash"`
}

type InspectionCollectionError struct {
	Node      string `json:"node,omitempty" yaml:"node,omitempty"`
	FactGroup string `json:"fact_group" yaml:"fact_group"`
	Source    string `json:"source,omitempty" yaml:"source,omitempty"`
	Message   string `json:"message" yaml:"message"`
}

type InspectionDatabaseProbe struct {
	Scope            InspectionScope                 `json:"scope" yaml:"scope"`
	Schemas          []InspectionSchemaEvidence      `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Communication    InspectionCommunicationEvidence `json:"communication" yaml:"communication"`
	CollectionErrors []InspectionCollectionError     `json:"collection_errors,omitempty" yaml:"collection_errors,omitempty"`
}

type InspectionDHCPOptionEvidence struct {
	UUID                 string            `json:"uuid" yaml:"uuid"`
	CIDR                 string            `json:"cidr" yaml:"cidr"`
	ExternalIDs          map[string]string `json:"external_ids,omitempty" yaml:"external_ids,omitempty"`
	Ports                []string          `json:"ports" yaml:"ports"`
	ClasslessStaticRoute []string          `json:"classless_static_route,omitempty" yaml:"classless_static_route,omitempty"`
}

type InspectionNetworkProbe struct {
	Scope            InspectionScope                `json:"scope" yaml:"scope"`
	DHCPOptions      []InspectionDHCPOptionEvidence `json:"dhcp_options" yaml:"dhcp_options"`
	CollectionErrors []InspectionCollectionError    `json:"collection_errors,omitempty" yaml:"collection_errors,omitempty"`
}
