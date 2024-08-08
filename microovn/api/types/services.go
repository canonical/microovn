// Package types provides shared types and structs.
package types

import (
	"log"
)

// Services - Slice with Service records.
type Services []Service

// Service  - A service.
type Service struct {
	// Service - name of Service.
	Service string `json:"service" yaml:"service"`
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
