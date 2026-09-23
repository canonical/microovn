package facts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/canonical/microovn/microovn/api/types"
)

func TestParseServiceStates(t *testing.T) {
	input := strings.NewReader("Service Startup Current Notes\n" +
		"microovn.switch disabled inactive -\n" +
		"microovn.refresh-expiring-certs enabled inactive timer-activated\n" +
		"microovn.daemon enabled active -\n")

	got, err := parseServiceStates(input)
	if err != nil {
		t.Fatalf("parseServiceStates returned an error: %v", err)
	}
	want := []types.InspectionDaemonState{
		{Name: "microovn.daemon", Active: true, Enabled: true},
		{Name: "microovn.switch", Active: false, Enabled: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseServiceStates returned %#v, want %#v", got, want)
	}

	for _, test := range []struct {
		name        string
		row         string
		wantEnabled bool
		wantActive  bool
	}{
		{name: "enabled active", row: "service enabled active -", wantEnabled: true, wantActive: true},
		{name: "enabled inactive", row: "service enabled inactive -", wantEnabled: true, wantActive: false},
		{name: "disabled active", row: "service disabled active -", wantEnabled: false, wantActive: true},
		{name: "disabled inactive", row: "service disabled inactive -", wantEnabled: false, wantActive: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, err := parseServiceState(test.row)
			if err != nil {
				t.Fatalf("parseServiceState(%q) returned an error: %v", test.row, err)
			}
			if state.Enabled != test.wantEnabled || state.Active != test.wantActive {
				t.Fatalf("parseServiceState(%q) returned enabled=%t active=%t", test.row, state.Enabled, state.Active)
			}
		})
	}

	for _, test := range []struct {
		name string
		row  string
	}{
		{name: "missing state", row: "microovn.daemon enabled"},
		{name: "unknown startup", row: "microovn.daemon automatic active -"},
		{name: "unknown current", row: "microovn.daemon enabled activating -"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseServiceState(test.row)
			if err == nil {
				t.Fatalf("parseServiceState(%q) succeeded", test.row)
			}
		})
	}
}

func TestParseEnvironment(t *testing.T) {
	const alphaValue = "alpha-secret"
	const zuluValue = "zulu-secret"
	input := strings.NewReader("# generated environment\n" +
		"\n" +
		"ZULU=\"" + zuluValue + "\"\n" +
		"malformed\n" +
		"ALPHA=\"" + alphaValue + "\"\n" +
		"ALPHA=\"duplicate\"\n" +
		"UNQUOTED=value\n")

	got, err := parseEnvironment(input)
	if err != nil {
		t.Fatalf("parseEnvironment returned an error: %v", err)
	}
	alphaHash := sha256.Sum256([]byte(alphaValue))
	zuluHash := sha256.Sum256([]byte(zuluValue))
	want := []types.InspectionEnvironment{
		{Name: "ALPHA", Hash: hex.EncodeToString(alphaHash[:])},
		{Name: "ZULU", Hash: hex.EncodeToString(zuluHash[:])},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseEnvironment returned %#v, want %#v", got, want)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("failed to marshal environment evidence: %v", err)
	}
	for _, rawValue := range []string{alphaValue, zuluValue, "duplicate", "value"} {
		if strings.Contains(string(encoded), rawValue) {
			t.Fatalf("serialized environment contains raw value %q", rawValue)
		}
	}
}

func TestCollectNodeSnapshotPreservesPartialResults(t *testing.T) {
	validServices := func(context.Context) (io.Reader, error) {
		return strings.NewReader("Service Startup Current Notes\n" +
			"microovn.daemon enabled active -\n"), nil
	}
	validEnvironment := func() (io.Reader, error) {
		return strings.NewReader("OVN_TEST=\"value\"\n"), nil
	}

	tests := []struct {
		name             string
		services         func(context.Context) (io.Reader, error)
		environment      func() (io.Reader, error)
		wantDaemons      int
		wantEnvironments int
		wantErrorGroup   string
	}{
		{
			name: "service collection fails",
			services: func(context.Context) (io.Reader, error) {
				return nil, errors.New("services unavailable")
			},
			environment:      validEnvironment,
			wantEnvironments: 1,
			wantErrorGroup:   "daemons",
		},
		{
			name:     "environment collection fails",
			services: validServices,
			environment: func() (io.Reader, error) {
				return nil, errors.New("environment unavailable")
			},
			wantDaemons:    1,
			wantErrorGroup: "environment",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := CollectNodeSnapshot(
				context.Background(),
				"node1",
				"10.0.0.1",
				test.services,
				test.environment,
			)

			if snapshot.NodeName != "node1" || snapshot.Address != "10.0.0.1" {
				t.Fatalf("snapshot identity = %q/%q", snapshot.NodeName, snapshot.Address)
			}
			if len(snapshot.Daemons) != test.wantDaemons {
				t.Errorf("daemon count = %d, want %d", len(snapshot.Daemons), test.wantDaemons)
			}
			if len(snapshot.Environment) != test.wantEnvironments {
				t.Errorf("environment count = %d, want %d", len(snapshot.Environment), test.wantEnvironments)
			}
			if len(snapshot.Errors) != 1 || snapshot.Errors[0].FactGroup != test.wantErrorGroup {
				t.Fatalf("snapshot errors = %#v, want one %q error", snapshot.Errors, test.wantErrorGroup)
			}
		})
	}
}
