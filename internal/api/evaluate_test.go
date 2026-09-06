package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"featureflagservice/internal/store"
)

func evaluateRequest(t *testing.T, key, user string, setUser bool) *httptest.ResponseRecorder {
	t.Helper()
	h := NewHandlers(store.NewStore())
	url := "/flags/" + key + "/evaluate"
	if setUser {
		url += "?user=" + user
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.SetPathValue("key", key)
	rec := httptest.NewRecorder()
	h.EvaluateFlag(rec, req)
	return rec
}

func TestEvaluateFlagMissingUser(t *testing.T) {
	rec := evaluateRequest(t, "enabled", "", false)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing user, got %d", rec.Code)
	}
}

func TestEvaluateFlagEmptyUser(t *testing.T) {
	rec := evaluateRequest(t, "enabled", "", true)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty user, got %d", rec.Code)
	}
}

func TestEvaluateFlagUnknownFlag(t *testing.T) {
	rec := evaluateRequest(t, "unknown", "user-1", true)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown flag, got %d", rec.Code)
	}
}

func TestEvaluateFlagDisabled(t *testing.T) {
	got := evaluateFlag(store.Flag{Key: "f", Enabled: false, RolloutPercent: 100}, "user-1")
	if got {
		t.Fatalf("disabled flag must evaluate to false, got true")
	}
}

func TestEvaluateFlagRolloutZero(t *testing.T) {
	got := evaluateFlag(store.Flag{Key: "f", Enabled: true, RolloutPercent: 0}, "user-1")
	if got {
		t.Fatalf("rollout 0 must evaluate to false, got true")
	}
}

func TestEvaluateFlagRolloutHundred(t *testing.T) {
	got := evaluateFlag(store.Flag{Key: "f", Enabled: true, RolloutPercent: 100}, "user-1")
	if !got {
		t.Fatalf("rollout 100 must evaluate to true, got false")
	}
}

func TestEvaluateFlagDeterministic(t *testing.T) {
	flag := store.Flag{Key: "f", Enabled: true, RolloutPercent: 50}
	first := evaluateFlag(flag, "user-1")
	for i := 0; i < 100; i++ {
		if got := evaluateFlag(flag, "user-1"); got != first {
			t.Fatalf("evaluate not deterministic: first=%v got=%v", first, got)
		}
	}
}
