// Package types provides shared types and structs.
package types

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

// DisableServiceRequest defines structure of a request to disable OVN services on the node
type DisableServiceRequest struct {
	AllowDisableLastCentral bool `json:"allowDisableLastCentral"` // If set to true, MicroOVN will allow removal of the last ovn-central cluster member. Effectively removing the central cluster and its data.
}

// Services - Slice with Service records.
type Services []Service

// Service  - A service.
type Service struct {
	// Service - name of Service.
	Service SrvName `json:"service" yaml:"service"`
	// Location - location of Service.
	Location string `json:"location" yaml:"location"`
}

// WarningSet - a set of warnings on the desired service state.
type WarningSet struct {
	// EvenCentral - are there an even number of central services which is
	// inefficent due to how RAFT works.
	EvenCentral bool `json:"EvenCentral" yaml:"EvenCentral"`
	// FewCentral - are there not enough central services to handle one
	// node failure.
	FewCentral bool `json:"FewCentral" yaml:"FewCentral"`
}

// ServiceControlResponse (SCR) - a struct to return both a response and any
// warnings, usually used when interfacing with the service control functions.
type ServiceControlResponse struct {
	// Message - any output needed from the service control functions.
	Message string `json:"message" yaml:"message"`
	// Warnings - the set of warnings with the desired state of services.
	Warnings WarningSet `json:"warnings" yaml:"warnings"`
}

// PrettyPrint - Formats and prints contents of WarningSet object.
func (w WarningSet) PrettyPrint(verbose bool) {
	if w.EvenCentral {
		if verbose {
			log.Println("[central] Warning: Cluster with even number of members has same fault tolerance, but higher quorum requirements, than cluster with one less member.")
		} else {
			log.Println("[central] Warning: OVN Cluster has even number of members")
		}
	}

	if w.FewCentral {
		if verbose {
			log.Println("[central] Warning: Cluster with less than 3 nodes can't tolerate any node failures.")
		} else {
			log.Println("[central] Warning: OVN Cluster has critically few members")
		}
	}
}

// RegenerateEnvResponse is a structure that models response to requests for
// a environment file regeneration for all nodes
type RegenerateEnvResponse struct {
	Success bool     `json:"success" yaml:"success"` // True if this node regenerates its environment
	Errors  []string `json:"error"`                  // List of Errors
}

// PrettyPrint method formats and prints contents of RegenerateEnvResponse object
func (r *RegenerateEnvResponse) PrettyPrint() {
	var newEnvSuccess string
	if r.Success {
		newEnvSuccess = "Generated"
	} else {
		newEnvSuccess = "Not Generated"
	}

	fmt.Printf("New Environment: %s\n\n", newEnvSuccess)

	if len(r.Errors) != 0 {
		fmt.Println("\n[Errors]")
		for _, errMsg := range r.Errors {
			fmt.Println(errMsg)
		}
	}
}

// NewRegenerateEnvResponse returns pointer to initialized RegenerateEnvResponse object
func NewRegenerateEnvResponse() RegenerateEnvResponse {
	return RegenerateEnvResponse{
		Success: false,
		Errors:  make([]string, 0),
	}
}

// SrvName - string representation of a service.
type SrvName = string

const (
	// SrvChassis - string representation of chassis service.
	SrvChassis SrvName = "chassis"
	// SrvCentral - string representation of central service.
	SrvCentral SrvName = "central"
	// SrvSwitch - string representation of switch service.
	SrvSwitch SrvName = "switch"
	// SrvBgp - string representation of BGP service
	SrvBgp SrvName = "bgp"
)

// ServiceNames - slice containing all known SrvName strings.
var ServiceNames = []SrvName{SrvBgp, SrvChassis, SrvCentral, SrvSwitch}

// ExtraServiceConfig - structure containing optional extra configuration for enabling service
type ExtraServiceConfig struct {
	BgpConfig *ExtraBgpConfig `json:"bgpConfig,omitempty" yaml:"bgpConfig,omitempty"`
}

// ExtraBgpConfig holds extra config options that can be used when enabling BGP config
type ExtraBgpConfig struct {
	// ExternalConnection is comma separated list of <iface_name>:<ip4_cidr> values. "iface_name"
	// is a name of the physical interface that provides connectivity to the external network and
	// "ip4_cidr" is IPv4 address (e.g. 192.0.2.1/24) that should be assigned to a Logical Router
	// Port connected to the external network
	ExternalConnection string `json:"ext_iface,omitempty" yaml:"ext_iface,omitempty"`
	// Vrf is a VRF table ID into which the OVN will leak its routes
	Vrf string `json:"vrf,omitempty" yaml:"vrf,omitempty"`
	// Asn is an Autonomous System Number that will be used to set up BGP daemon
	Asn string `json:"asn,omitempty" yaml:"asn,omitempty"`
}

// BgpExternalConnection represents a parsed structure from ExtraBgpConfig.ExternalConnection string.
type BgpExternalConnection struct {
	// Iface is a name of the physical interface that provides external connectivity
	Iface string
	// IPAddress is an IP that is assigned to the Logical Router Port connected to the external network
	IPAddress net.IP
	// IPMask is network mask assigned to the Logical Router Port connected to the external network
	IPMask net.IPMask
}

// FromMap initializes ExtraBgpConfig structure from the provided map of string keys and string values.
// This functions also validates the resulting structure and returns error if the validation fails.
func (bgpConf *ExtraBgpConfig) FromMap(rawConfig map[string]string) error {
	for key, value := range rawConfig {
		if key == "ext_connection" {
			bgpConf.ExternalConnection = value
			continue
		}
		if key == "vrf" {
			bgpConf.Vrf = value
			continue
		}
		if key == "asn" {
			bgpConf.Asn = value
			continue
		}
		return fmt.Errorf("unknown BGP config option: %s", key)
	}
	return bgpConf.Validate()
}

// Validate ensures that all required fields of ExtraBgpConfig are present and that they have
// correct types and values.
func (bgpConf *ExtraBgpConfig) Validate() error {
	if bgpConf.Vrf == "" {
		return fmt.Errorf("option 'vrf' is rquired")
	}

	_, err := strconv.Atoi(bgpConf.Vrf)
	if err != nil {
		return fmt.Errorf("option 'vrf' is not a number: %s", bgpConf.Vrf)
	}

	_, err = strconv.Atoi(bgpConf.Asn)
	if err != nil && bgpConf.Asn != "" {
		return fmt.Errorf("option 'asn' is not a number: %s", bgpConf.Asn)
	}

	extConnections, err := bgpConf.ParseExternalConnection()
	if err != nil {
		return fmt.Errorf("failed to parse connection string option: %s", err)
	}

	if len(extConnections) == 0 {
		return fmt.Errorf("external connections have to be set")
	}

	return nil
}

// ParseExternalConnection parses ExtraBgpConfig.ExternalConnection string into list of BgpExternalConnection
// instances.
func (bgpConf *ExtraBgpConfig) ParseExternalConnection() ([]BgpExternalConnection, error) {
	parsedConnections := make([]BgpExternalConnection, 0)
	for _, extConn := range strings.Split(bgpConf.ExternalConnection, ",") {
		ifaceName, cidr, found := strings.Cut(extConn, ":")
		if !found {
			return nil, fmt.Errorf("connection string requires format '<interface_name>:<ipv4_cidr>': %s", extConn)
		}

		ipAddr, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid IPv4 CIDR notation: %s", cidr)
		}
		parsedConnections = append(parsedConnections, BgpExternalConnection{
			Iface:     ifaceName,
			IPAddress: ipAddr,
			IPMask:    ipNet.Mask,
		})
	}

	return parsedConnections, nil
}

// CheckValidService - checks whether the string in "service" is in fact a
// known and valid service name.
func CheckValidService(service string) bool {
	for _, s := range ServiceNames {
		if s == service {
			return true
		}
	}
	return false
}
