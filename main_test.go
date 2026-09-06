package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return newHandler(log.New(io.Discard, "", 0))
}

func TestHealthz(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET /healthz body is not valid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("GET /healthz status field = %q, want %q", body["status"], "ok")
	}
}

// TestRoutesAreRegistered asserts every declared route is wired to the mux
// under its agreed path and verb. It checks reachability (not a 404), never
// the temporary answer of a stub handler, so it stays valid once the owning
// tickets replace the stubs with real implementations.
func TestRoutesAreRegistered(t *testing.T) {
	h := newTestHandler(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/healthz"},
		{http.MethodPost, "/flags"},
		{http.MethodGet, "/flags"},
		{http.MethodGet, "/flags/example"},
		{http.MethodPut, "/flags/example"},
		{http.MethodDelete, "/flags/example"},
		{http.MethodGet, "/flags/example/evaluate"},
	}

	for _, r := range routes {
		req := httptest.NewRequest(r.method, r.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Errorf("%s %s is not registered (answered 404)", r.method, r.path)
		}
	}
}

func TestWrongMethodReturns405JSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /healthz status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("405 body is not valid JSON: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("405 body has no error field: %v", body)
	}
}

func TestUnknownPathReturns404JSON(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /nope status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("404 body is not valid JSON: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("404 body has no error field: %v", body)
	}
}
