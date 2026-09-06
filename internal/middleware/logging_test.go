package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingLogsMethodPathStatus(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, "", 0)

	handler := Logging(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/flags", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	line := buf.String()
	if !strings.Contains(line, "POST") {
		t.Errorf("log line missing method, got: %q", line)
	}
	if !strings.Contains(line, "/flags") {
		t.Errorf("log line missing path, got: %q", line)
	}
	if !strings.Contains(line, "201") {
		t.Errorf("log line missing status, got: %q", line)
	}
}

func TestLoggingDefaultStatus200(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, "", 0)

	handler := Logging(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("underlying status = %d, want 200", rr.Code)
	}
	if !strings.Contains(buf.String(), "200") {
		t.Errorf("log line missing default 200 status, got: %q", buf.String())
	}
}

func TestLoggingExcludesUserQueryParam(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, "", 0)

	handler := Logging(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/flags/myflag/evaluate?user=alice", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	line := buf.String()
	if strings.Contains(line, "alice") {
		t.Errorf("log line leaks user id, got: %q", line)
	}
	if strings.Contains(line, "user") {
		t.Errorf("log line leaks query parameter, got: %q", line)
	}
	if strings.Contains(line, "?") {
		t.Errorf("log line contains query string, got: %q", line)
	}
}
