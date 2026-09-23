package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInspectionSnapshotEndpointContract(t *testing.T) {
	if InspectionSnapshotEndpoint.Path != "inspect/snapshot" {
		t.Fatalf("endpoint path = %q, want %q", InspectionSnapshotEndpoint.Path, "inspect/snapshot")
	}
	if InspectionSnapshotEndpoint.Get.Handler == nil {
		t.Fatal("snapshot endpoint has no GET handler")
	}
	if InspectionSnapshotEndpoint.Get.AllowUntrusted {
		t.Fatal("snapshot endpoint allows untrusted requests")
	}
	if InspectionSnapshotEndpoint.Get.ProxyTarget {
		t.Fatal("snapshot endpoint proxies requests")
	}

	registered := false
	for _, resource := range Server["microovn"].Resources {
		for _, endpoint := range resource.Endpoints {
			if endpoint.Path == InspectionSnapshotEndpoint.Path {
				registered = true
				break
			}
		}
	}
	if !registered {
		t.Fatal("snapshot endpoint is not registered")
	}
}

func TestInspectionDatabaseEndpointContract(t *testing.T) {
	if InspectionDatabaseEndpoint.Path != "inspect/database" {
		t.Fatalf("endpoint path = %q, want %q", InspectionDatabaseEndpoint.Path, "inspect/database")
	}
	if InspectionDatabaseEndpoint.Get.Handler == nil {
		t.Fatal("database endpoint has no GET handler")
	}
	if InspectionDatabaseEndpoint.Get.AllowUntrusted {
		t.Fatal("database endpoint allows untrusted requests")
	}
	if InspectionDatabaseEndpoint.Get.ProxyTarget {
		t.Fatal("database endpoint proxies requests")
	}

	registered := false
	for _, resource := range Server["microovn"].Resources {
		for _, endpoint := range resource.Endpoints {
			if endpoint.Path == InspectionDatabaseEndpoint.Path {
				registered = true
				break
			}
		}
	}
	if !registered {
		t.Fatal("database endpoint is not registered")
	}
}

func TestInspectionNetworkEndpointContract(t *testing.T) {
	if InspectionNetworkEndpoint.Path != "inspect/network" {
		t.Fatalf("endpoint path = %q, want %q", InspectionNetworkEndpoint.Path, "inspect/network")
	}
	if InspectionNetworkEndpoint.Get.Handler == nil {
		t.Fatal("network endpoint has no GET handler")
	}
	if InspectionNetworkEndpoint.Get.AllowUntrusted {
		t.Fatal("network endpoint allows untrusted requests")
	}
	if InspectionNetworkEndpoint.Get.ProxyTarget {
		t.Fatal("network endpoint proxies requests")
	}

	registered := false
	for _, resource := range Server["microovn"].Resources {
		for _, endpoint := range resource.Endpoints {
			if endpoint.Path == InspectionNetworkEndpoint.Path {
				registered = true
				break
			}
		}
	}
	if !registered {
		t.Fatal("network endpoint is not registered")
	}
}

func TestInspectionDatabaseGetRejectsInvalidScope(t *testing.T) {
	for _, target := range []string{
		"/1.0/inspect/database",
		"/1.0/inspect/database?scope=invalid",
	} {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			recorder := httptest.NewRecorder()

			response := inspectionDatabaseGet(nil, request)
			if err := response.Render(recorder, request); err != nil {
				t.Fatalf("failed to render response: %v", err)
			}
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
			}
		})
	}
}
