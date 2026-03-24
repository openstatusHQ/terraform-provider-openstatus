package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
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
	if c.restBaseURL != DefaultRESTBaseURL {
		t.Errorf("expected default REST base URL %q, got %q", DefaultRESTBaseURL, c.restBaseURL)
	}
}

func TestNew_CustomBaseURL_DerivesRESTBaseURL(t *testing.T) {
	c := New("https://custom.example.com/rpc", "key")
	if c.baseURL != "https://custom.example.com/rpc" {
		t.Errorf("expected custom base URL, got %q", c.baseURL)
	}
	if c.restBaseURL != "https://custom.example.com/v1" {
		t.Errorf("expected derived REST base URL %q, got %q", "https://custom.example.com/v1", c.restBaseURL)
	}
}

func TestDoREST_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/page/42" {
			t.Errorf("expected path /page/42, got %s", r.URL.Path)
		}
		if r.Header.Get("x-openstatus-key") != "test-key" {
			t.Errorf("expected api key header, got %q", r.Header.Get("x-openstatus-key"))
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 42, "title": "Test"})
	}))
	defer server.Close()

	c := New("", "test-key")
	c.restBaseURL = server.URL

	var resp map[string]interface{}
	err := c.DoREST(context.Background(), http.MethodGet, "/page/42", nil, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp["title"] != "Test" {
		t.Errorf("expected title=Test, got %v", resp["title"])
	}
}

func TestDoREST_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var reqBody map[string]string
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["title"] != "New Page" {
			t.Errorf("expected title=New Page, got %q", reqBody["title"])
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": 1, "title": "New Page"})
	}))
	defer server.Close()

	c := New("", "test-key")
	c.restBaseURL = server.URL

	var resp map[string]interface{}
	err := c.DoREST(context.Background(), http.MethodPost, "/page", map[string]string{"title": "New Page"}, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp["title"] != "New Page" {
		t.Errorf("expected title=New Page, got %v", resp["title"])
	}
}

func TestDoREST_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "not_found",
			"message": "page not found",
		})
	}))
	defer server.Close()

	c := New("", "test-key")
	c.restBaseURL = server.URL

	err := c.DoREST(context.Background(), http.MethodGet, "/page/999", nil, nil)
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
