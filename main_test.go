package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
// under its agreed path and verb. A real handler may legitimately answer 404
// (e.g. a flag key that does not exist yet), so a bare 404 alone is not proof
// the route is missing: only the mux's own fallback ({"error":"not found"})
// means the path was never registered.
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
			var body map[string]string
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			if body["error"] == "not found" {
				t.Errorf("%s %s is not registered (answered 404)", r.method, r.path)
			}
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

// TestServerServesHealthz starts the real http.Server (with its timeouts) on
// an ephemeral port and probes GET /healthz over an actual TCP connection,
// proving the server boot and health handler work at runtime, not just in
// handler-isolation tests.
func TestServerServesHealthz(t *testing.T) {
	srv := newServer("127.0.0.1:0", log.New(io.Discard, "", 0))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	resp, err := http.Get("http://" + ln.Addr().String() + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET /healthz body is not valid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("GET /healthz status field = %q, want %q", body["status"], "ok")
	}
}

func TestServerTimeouts(t *testing.T) {
	srv := newServer(":8080", log.New(io.Discard, "", 0))
	if srv.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout = %v, want 5s", srv.ReadTimeout)
	}
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 5s", srv.ReadHeaderTimeout)
	}
	if srv.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout = %v, want 10s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout = %v, want 60s", srv.IdleTimeout)
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
