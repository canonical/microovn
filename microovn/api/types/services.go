// Package types provides shared types and structs.
package types

import (
	"fmt"
	"log"
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
)

// ServiceNames - slice containing all known SrvName strings.
var ServiceNames = []SrvName{SrvChassis, SrvCentral, SrvSwitch}

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
