// Package types provides shared types and structs.
package types

// Services - Slice with Service records.
type Services []Service

// Service  - A service.
type Service struct {
	// Service - name of Service
	Service string `json:"service" yaml:"service"`
	// Location - location of Service
	Location string `json:"location" yaml:"location"`
}
