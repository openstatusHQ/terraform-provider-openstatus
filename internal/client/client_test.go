package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"connectrpc.com/connect"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"connect not found", connect.NewError(connect.CodeNotFound, errors.New("gone")), true},
		{"connect other code", connect.NewError(connect.CodeInvalidArgument, errors.New("bad")), false},
		{"plain error", errors.New("boom"), false},
	}

	for _, tt := range tests {
		if got := IsNotFound(tt.err); got != tt.expected {
			t.Errorf("IsNotFound(%s) = %v, want %v", tt.name, got, tt.expected)
		}
	}
}

func TestNew_AppendsProcedureToBaseURL(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := New(server.URL, "key")
	if _, err := c.Monitor.GetMonitor(context.Background(), connect.NewRequest(&monitorv1.GetMonitorRequest{})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/openstatus.monitor.v1.MonitorService/GetMonitor" {
		t.Errorf("path = %q, want the Connect procedure appended to the base URL", gotPath)
	}
}

func TestNew_TypedClientsWired(t *testing.T) {
	c := New("", "key")
	if c.Monitor == nil || c.Notification == nil || c.StatusPage == nil || c.PrivateLocation == nil {
		t.Fatal("expected all typed service clients to be constructed")
	}
}

func TestNew_AuthInterceptorSetsHeader(t *testing.T) {
	var gotKey, gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-openstatus-key")
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := New(server.URL, "test-key")
	if _, err := c.Monitor.GetMonitor(context.Background(), connect.NewRequest(&monitorv1.GetMonitorRequest{})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "test-key" {
		t.Errorf("expected the api key header to be set by the interceptor, got %q", gotKey)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json (WithProtoJSON)", gotContentType)
	}
}

func TestClient_MapsNotFoundResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"not_found","message":"resource not found"}`))
	}))
	defer server.Close()

	c := New(server.URL, "key")
	_, err := c.Monitor.GetMonitor(context.Background(), connect.NewRequest(&monitorv1.GetMonitorRequest{}))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound = false for %v, want true", err)
	}
}
