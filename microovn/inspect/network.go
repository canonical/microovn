package inspect

import (
	"context"
	"net/netip"
	"strings"

	"github.com/canonical/microovn/microovn/api/types"
)

const metadataIPv4CIDR = "169.254.169.254/32"

type NetworkCheck struct{}

func (NetworkCheck) Name() string {
	return "network"
}

func (NetworkCheck) Run(_ context.Context, input Input) []types.InspectionResult {
	if !input.ExecutionContext.Authoritative {
		return []types.InspectionResult{nonAuthoritativeNetworkResult()}
	}

	var collectionError strings.Builder
	for _, collectionErr := range input.CollectionErrors {
		if collectionErr.FactGroup != "network" {
			continue
		}
		if collectionError.Len() > 0 {
			collectionError.WriteString("; ")
		}
		collectionError.WriteString(collectionErr.Message)
	}
	for _, collectionErr := range input.Network.CollectionErrors {
		if collectionError.Len() > 0 {
			collectionError.WriteString("; ")
		}
		collectionError.WriteString(collectionErr.Message)
	}
	if collectionError.Len() > 0 {
		return []types.InspectionResult{{
			ID:              "network",
			Category:        "network",
			Status:          types.InspectionStatusUnknown,
			Summary:         "Network state is unavailable",
			CollectionError: collectionError.String(),
		}}
	}

	details, applicableOptions := networkDetails(input.Network)
	if len(details) > 0 {
		return []types.InspectionResult{{
			ID:          "dhcp-metadata-route",
			Category:    "network",
			Status:      types.InspectionStatusFail,
			Summary:     "DHCP options are missing a valid metadata route",
			Details:     details,
			Remediation: "Restore the metadata route in the affected DHCP options.",
		}}
	}

	summary := "DHCP metadata routes are configured"
	if applicableOptions == 0 {
		summary = "No applicable DHCPv4 options were found"
	}

	return []types.InspectionResult{{
		ID:       "network",
		Category: "network",
		Status:   types.InspectionStatusPass,
		Summary:  summary,
	}}
}

func networkDetails(network types.InspectionNetworkProbe) ([]types.InspectionDetail, int) {
	var details []types.InspectionDetail
	applicableOptions := 0

	for _, option := range network.DHCPOptions {
		prefix, err := netip.ParsePrefix(option.CIDR)
		if len(option.Ports) == 0 || err != nil || !prefix.Addr().Is4() {
			continue
		}
		applicableOptions++

		if hasMetadataRoute(option.ClasslessStaticRoute) {
			continue
		}

		details = append(details, types.InspectionDetail{
			ID:      "dhcp-metadata-route",
			Status:  types.InspectionStatusFail,
			Summary: "DHCP options are missing a valid metadata route",
			Data: map[string]string{
				"uuid":  option.UUID,
				"cidr":  option.CIDR,
				"ports": strings.Join(option.Ports, ","),
			},
		})
	}

	return details, applicableOptions
}

func hasMetadataRoute(routes []string) bool {
	for _, route := range routes {
		destination, nextHop, found := strings.Cut(route, ",")
		if !found || strings.TrimSpace(destination) != metadataIPv4CIDR {
			continue
		}

		address, err := netip.ParseAddr(strings.TrimSpace(nextHop))
		if err == nil && address.Is4() {
			return true
		}
	}

	return false
}

func nonAuthoritativeNetworkResult() types.InspectionResult {
	return types.InspectionResult{
		ID:          "network",
		Category:    "network",
		Status:      types.InspectionStatusUnknown,
		Summary:     "Network state cannot be determined from a non-voting member",
		Remediation: "Run the inspection on a voting member.",
	}
}
