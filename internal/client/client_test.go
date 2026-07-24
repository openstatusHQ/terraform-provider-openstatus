package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"connectrpc.com/connect"
)

func TestDo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-openstatus-key") != "test-key" {
			t.Errorf("expected api key header, got %q", r.Header.Get("x-openstatus-key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected json content type, got %q", r.Header.Get("Content-Type"))
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var reqBody map[string]string
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["name"] != "test" {
			t.Errorf("expected name=test, got %q", reqBody["name"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer server.Close()

	c := New(server.URL, "test-key")

	var resp map[string]string
	err := c.Do(context.Background(), "/test", map[string]string{"name": "test"}, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp["id"] != "123" {
		t.Errorf("expected id=123, got %q", resp["id"])
	}
}

func TestDo_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "invalid_argument",
			"message": "name is required",
		})
	}))
	defer server.Close()

	c := New(server.URL, "test-key")

	err := c.Do(context.Background(), "/test", map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "invalid_argument" {
		t.Errorf("expected code=invalid_argument, got %q", apiErr.Code)
	}
}

func TestDo_NotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "not_found",
			"message": "resource not found",
		})
	}))
	defer server.Close()

	c := New(server.URL, "test-key")

	err := c.Do(context.Background(), "/test", map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "not_found" {
		t.Errorf("expected code=not_found, got %q", apiErr.Code)
	}
}

func TestDo_NonJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	c := New(server.URL, "test-key")

	err := c.Do(context.Background(), "/test", map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDo_NilResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer server.Close()

	c := New(server.URL, "test-key")

	err := c.Do(context.Background(), "/test", map[string]string{"id": "1"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_DefaultBaseURL(t *testing.T) {
	c := New("", "key")
	if c.baseURL != DefaultBaseURL {
		t.Errorf("expected default base URL, got %q", c.baseURL)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"legacy not_found", &APIError{Code: "not_found"}, true},
		{"legacy other code", &APIError{Code: "invalid_argument"}, false},
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

func TestNew_TypedClientsWired(t *testing.T) {
	c := New("", "key")
	if c.Monitor == nil || c.Notification == nil || c.StatusPage == nil || c.PrivateLocation == nil {
		t.Fatal("expected all typed service clients to be constructed")
	}
}

func TestNew_AuthInterceptorSetsHeader(t *testing.T) {
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-openstatus-key")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := New(server.URL, "test-key")
	_, err := c.Monitor.GetMonitor(context.Background(), connect.NewRequest(&monitorv1.GetMonitorRequest{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "test-key" {
		t.Errorf("expected api key header to be set by the interceptor, got %q", gotKey)
	}
}
