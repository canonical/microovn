package client

import (
	"context"
	"testing"

	"github.com/canonical/microovn/microovn/api/types"
)

func TestGetDatabaseProbeRejectsInvalidScope(t *testing.T) {
	_, err := GetDatabaseProbe(context.Background(), nil, types.InspectionScope("invalid"))
	if err == nil {
		t.Fatal("GetDatabaseProbe returned nil error for invalid scope")
	}
}
