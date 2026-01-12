package dpu

import "testing"

func TestFindDPUDevlink(t *testing.T) {
	tests := []struct {
		name     string
		input    DevlinkJSON
		expected string
	}{
		{
			name: "single pcipf controller 0",
			input: DevlinkJSON{
				Port: map[string]DevlinkPort{
					"pci/0000:03:14.0": {
						Flavour:    "pcipf",
						Controller: 0,
					},
				},
			},
			expected: "0000:03:14.0",
		},
		{
			name: "controller non-zero ignored",
			input: DevlinkJSON{
				Port: map[string]DevlinkPort{
					"pci/0000:03:14.0": {
						Flavour:    "pcipf",
						Controller: 1,
					},
				},
			},
			expected: "",
		},
		{
			name: "wrong flavour ignored",
			input: DevlinkJSON{
				Port: map[string]DevlinkPort{
					"pci/0000:03:14.0": {
						Flavour:    "pcisf",
						Controller: 0,
					},
				},
			},
			expected: "",
		},
		{
			name: "multiple ports first valid match wins",
			input: DevlinkJSON{
				Port: map[string]DevlinkPort{
					"pci/0000:06:12.0": {
						Flavour:    "pcisf",
						Controller: 0,
					},
					"pci/0000:11:11.0": {
						Flavour:    "pcipf",
						Controller: 0,
					},
					"pci/0000:03:14.0": {
						Flavour:    "pcipf",
						Controller: 1,
					},
				},
			},
			expected: "0000:11:11.0",
		},
		{
			name: "empty port map",
			input: DevlinkJSON{
				Port: map[string]DevlinkPort{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findDPUDevlink(tt.input)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseLspciSerial(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "serial present",
			input: `
00:00.0 Host bridge: Canonical Device 413
[SN] Serial number: VRSKA612413
Capabilities: [42] Post Scratch Reset
`,
			expected: "VRSKA612413",
		},
		{
			name: "serial with extra whitespace",
			input: `
[SN] Serial number:   JTRUANT102506
`,
			expected: "JTRUANT102506",
		},
		{
			name: "serial not present",
			input: `
00:00.0 Host bridge: Canonical Device 413
Capabilities: [42] Post Scratch Reset
`,
			expected: "",
		},
		{
			name:     "empty output",
			input:    "",
			expected: "",
		},
		{
			name: "multiple serial lines first wins",
			input: `
[SN] Serial number: FIRST413
[SN] Serial number: SECOND612
`,
			expected: "FIRST413",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLspciSerial(tt.input)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
