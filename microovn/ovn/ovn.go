// Package ovn hosts most of MicroOVN's life cycle management for OVN.
package ovn

import (
	"fmt"
	"strings"
)

type requestedServices struct {
	Central bool
	Chassis bool
	Switch  bool
}

func newRequestedServices(initString string) (requestedServices, error) {
	var err error
	newInstance := requestedServices{
		Central: false,
		Chassis: false,
		Switch:  false,
	}

	for _, service := range strings.Split(initString, ",") {
		switch service {
		case "central":
			newInstance.Central = true
		case "chassis":
			newInstance.Chassis = true
		case "switch":
			newInstance.Switch = true
		default:
			err = fmt.Errorf("invalid service requested: %s", service)
		}
	}
	return newInstance, err
}
