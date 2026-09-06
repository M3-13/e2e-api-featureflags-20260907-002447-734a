package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflagservice/internal/store"
)

func newTestHandlers() *Handlers {
	return NewHandlers(store.NewStore())
}

func doRequest(h *Handlers, method, path, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	if strings.HasPrefix(path, "/flags/") && method != http.MethodPost {
		r.SetPathValue("key", strings.TrimPrefix(path, "/flags/"))
	}
	w := httptest.NewRecorder()
	switch method {
	case http.MethodPost:
		h.CreateFlag(w, r)
	case http.MethodGet:
		if strings.HasPrefix(path, "/flags/") {
			h.GetFlag(w, r)
		} else {
			h.ListFlags(w, r)
		}
	case http.MethodPut:
		h.UpdateFlag(w, r)
	case http.MethodDelete:
		h.DeleteFlag(w, r)
	}
	return w
}

func decodeFlag(t *testing.T, body []byte) store.Flag {
	t.Helper()
	var f store.Flag
	if err := json.Unmarshal(body, &f); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	return f
}

func decodeError(t *testing.T, body []byte) string {
	t.Helper()
	var e map[string]string
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("error response is not valid JSON: %v", err)
	}
	return e["error"]
}

func TestCreateFlag(t *testing.T) {
	h := newTestHandlers()

	w := doRequest(h, http.MethodPost, "/flags",
		`{"key":"my.flag","enabled":true,"description":"hello","rollout_percent":42}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	f := decodeFlag(t, w.Body.Bytes())
	if f.Key != "my.flag" || !f.Enabled || f.Description != "hello" || f.RolloutPercent != 42 {
		t.Fatalf("unexpected flag: %+v", f)
	}
}

func TestCreateFlagRolloutDefaultsToZero(t *testing.T) {
	h := newTestHandlers()
	w := doRequest(h, http.MethodPost, "/flags", `{"key":"a","enabled":false}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	f := decodeFlag(t, w.Body.Bytes())
	if f.RolloutPercent != 0 {
		t.Fatalf("rollout_percent = %d, want 0", f.RolloutPercent)
	}
}

func TestCreateFlagValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty key", `{"key":"","enabled":true}`},
		{"missing key", `{"enabled":true}`},
		{"invalid key chars", `{"key":"bad key!","enabled":true}`},
		{"missing enabled", `{"key":"a"}`},
		{"rollout too low", `{"key":"a","enabled":true,"rollout_percent":-1}`},
		{"rollout too high", `{"key":"a","enabled":true,"rollout_percent":101}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandlers()
			w := doRequest(h, http.MethodPost, "/flags", tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
			}
			if decodeError(t, w.Body.Bytes()) == "" {
				t.Fatalf("expected JSON error object, got %s", w.Body.String())
			}
		})
	}
}

func TestCreateFlagDuplicate(t *testing.T) {
	h := newTestHandlers()
	body := `{"key":"dup","enabled":true}`

	if w := doRequest(h, http.MethodPost, "/flags", body); w.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want %d", w.Code, http.StatusCreated)
	}
	w := doRequest(h, http.MethodPost, "/flags", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want %d; body=%s", w.Code, http.StatusConflict, w.Body.String())
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}

func TestCreateFlagBodyTooLarge(t *testing.T) {
	h := newTestHandlers()
	body := `{"key":"big","enabled":true,"description":"` + strings.Repeat("a", 2<<20) + `"}`
	w := doRequest(h, http.MethodPost, "/flags", body)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}

func TestCreateFlagLimit(t *testing.T) {
	original := store.MaxFlags
	store.MaxFlags = 1
	defer func() { store.MaxFlags = original }()

	h := newTestHandlers()

	if w := doRequest(h, http.MethodPost, "/flags", `{"key":"one","enabled":true}`); w.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want %d; body=%s", w.Code, http.StatusCreated, w.Body.String())
	}

	w := doRequest(h, http.MethodPost, "/flags", `{"key":"two","enabled":true}`)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("limit status = %d, want %d; body=%s", w.Code, http.StatusTooManyRequests, w.Body.String())
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}

func TestListFlagsEmpty(t *testing.T) {
	h := newTestHandlers()
	w := doRequest(h, http.MethodGet, "/flags", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var flags []store.Flag
	if err := json.Unmarshal(w.Body.Bytes(), &flags); err != nil {
		t.Fatalf("response is not a JSON array: %v (%s)", err, w.Body.String())
	}
	if len(flags) != 0 {
		t.Fatalf("expected empty array, got %+v", flags)
	}
}

func TestListFlags(t *testing.T) {
	h := newTestHandlers()
	doRequest(h, http.MethodPost, "/flags", `{"key":"b","enabled":true}`)
	doRequest(h, http.MethodPost, "/flags", `{"key":"a","enabled":false}`)

	w := doRequest(h, http.MethodGet, "/flags", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var flags []store.Flag
	if err := json.Unmarshal(w.Body.Bytes(), &flags); err != nil {
		t.Fatalf("response is not a JSON array: %v", err)
	}
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %+v", flags)
	}
	keys := map[string]bool{}
	for _, f := range flags {
		keys[f.Key] = true
	}
	if !keys["a"] || !keys["b"] {
		t.Fatalf("expected keys a and b, got %+v", flags)
	}
}

func TestGetFlag(t *testing.T) {
	h := newTestHandlers()
	doRequest(h, http.MethodPost, "/flags", `{"key":"found","enabled":true,"description":"d"}`)

	w := doRequest(h, http.MethodGet, "/flags/found", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	f := decodeFlag(t, w.Body.Bytes())
	if f.Key != "found" || !f.Enabled || f.Description != "d" {
		t.Fatalf("unexpected flag: %+v", f)
	}
}

func TestGetFlagInvalidKey(t *testing.T) {
	h := newTestHandlers()
	w := doRequest(h, http.MethodGet, "/flags/bad!key", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}

func TestGetFlagNotFound(t *testing.T) {
	h := newTestHandlers()
	w := doRequest(h, http.MethodGet, "/flags/nope", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}

func TestUpdateFlag(t *testing.T) {
	h := newTestHandlers()
	doRequest(h, http.MethodPost, "/flags", `{"key":"up","enabled":true,"description":"old","rollout_percent":10}`)

	w := doRequest(h, http.MethodPut, "/flags/up",
		`{"enabled":false,"description":"new","rollout_percent":80}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	f := decodeFlag(t, w.Body.Bytes())
	if f.Key != "up" || f.Enabled || f.Description != "new" || f.RolloutPercent != 80 {
		t.Fatalf("unexpected updated flag: %+v", f)
	}
}

func TestUpdateFlagNotFound(t *testing.T) {
	h := newTestHandlers()
	w := doRequest(h, http.MethodPut, "/flags/nope", `{"enabled":true}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}

func TestUpdateFlagInvalidKey(t *testing.T) {
	h := newTestHandlers()
	w := doRequest(h, http.MethodPut, "/flags/bad!key", `{"enabled":true}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}

func TestUpdateFlagValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing enabled", `{}`},
		{"rollout too low", `{"enabled":true,"rollout_percent":-1}`},
		{"rollout too high", `{"enabled":true,"rollout_percent":101}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandlers()
			doRequest(h, http.MethodPost, "/flags", `{"key":"up","enabled":true}`)
			w := doRequest(h, http.MethodPut, "/flags/up", tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

func TestDeleteFlag(t *testing.T) {
	h := newTestHandlers()
	doRequest(h, http.MethodPost, "/flags", `{"key":"del","enabled":true}`)

	w := doRequest(h, http.MethodDelete, "/flags/del", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNoContent, w.Body.String())
	}

	gw := doRequest(h, http.MethodGet, "/flags/del", "")
	if gw.Code != http.StatusNotFound {
		t.Fatalf("after delete GET status = %d, want %d", gw.Code, http.StatusNotFound)
	}
}

func TestDeleteFlagNotFound(t *testing.T) {
	h := newTestHandlers()
	w := doRequest(h, http.MethodDelete, "/flags/nope", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusNotFound, w.Body.String())
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}

func TestDeleteFlagInvalidKey(t *testing.T) {
	h := newTestHandlers()
	w := doRequest(h, http.MethodDelete, "/flags/bad!key", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if decodeError(t, w.Body.Bytes()) == "" {
		t.Fatalf("expected JSON error object, got %s", w.Body.String())
	}
}
