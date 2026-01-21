package environment

import (
	"net/url"
	"testing"

	"github.com/canonical/microcluster/v3/state"
)

const localNodeIPv4 = "10.0.0.1"
const remoteNodeIPv4 = "10.0.0.2"
const localNodeIPv6 = "fe80::1"
const remoteNodeIPv6 = "fe80::2"

type MockState struct {
	localNodeIP string
	state.State
}

func (ms MockState) Address() *url.URL {
	netURL := url.URL{
		Scheme: "https",
		Host:   ms.localNodeIP + ":6443",
		Path:   "/1.0",
	}
	return &netURL
}

func TestUnexported_initialNbSbHost(t *testing.T) {
	testCases := []struct {
		hostIP     string
		isIPv6     bool
		centralIps []string
		expected   string
	}{
		// IPv4 multi-node cluster: Expect initial to differ from local node
		{
			hostIP:     localNodeIPv4,
			isIPv6:     false,
			centralIps: []string{localNodeIPv4, remoteNodeIPv4},
			expected:   remoteNodeIPv4,
		},
		// IPv4 single-node: Expect initial to match the local node
		{
			hostIP:     localNodeIPv4,
			isIPv6:     false,
			centralIps: []string{localNodeIPv4},
			expected:   localNodeIPv4,
		},
		// IPv6 multi-node cluster: Expect initial to differ from local node
		{
			hostIP:     localNodeIPv6,
			isIPv6:     true,
			centralIps: []string{localNodeIPv6, remoteNodeIPv6},
			expected:   remoteNodeIPv6,
		},
		// IPv4 single-node: Expect initial to match the local node
		{
			hostIP:     localNodeIPv6,
			isIPv6:     true,
			centralIps: []string{localNodeIPv6},
			expected:   localNodeIPv6,
		},
	}

	for _, tc := range testCases {
		initialHost, err := initialNbSbHost(MockState{localNodeIP: tc.hostIP}, tc.centralIps)
		if err != nil {
			t.Errorf("Failed to get initial host: %s", err)
		}
		expectedIP := tc.expected
		if tc.isIPv6 {
			expectedIP = "[" + tc.expected + "]"
		}
		if initialHost != expectedIP {
			t.Errorf("Expected initial host to be %s, got %s", expectedIP, initialHost)
		}
	}
}
