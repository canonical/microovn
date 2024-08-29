// Package version provides shared version information.
package version

import "strings"

// MicroOvnVersion contains version of MicroOVN (set at build time)
var MicroOvnVersion string

// OvnVersion contains version of 'ovn' package used to build MicroOVN (set at build time)
var OvnVersion string

// OvsVersion contains version of 'openvswitch' package used to build MicroOVN (set at build time)
var OvsVersion string

// MajorVersion extracts the major version of the supplied "version" argument.
func MajorVersion(version string) string {
	versionSlice := strings.Split(version, ".")
	return strings.Join(versionSlice[:min(2, len(versionSlice))], ".")
}
