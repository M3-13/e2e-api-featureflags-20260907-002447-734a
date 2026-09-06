package api

import (
	"net/http"

	"featureflagservice/internal/evaluate"
	"featureflagservice/internal/store"
)

func (h *Handlers) EvaluateFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	user := r.Header.Get("X-User-ID")
	if user == "" {
		writeError(w, http.StatusBadRequest, "user is required")
		return
	}
	if !validKey(key) {
		writeError(w, http.StatusBadRequest, "key may only contain [A-Za-z0-9._-]")
		return
	}

	flag, ok := h.store.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"result": evaluateFlag(flag, user)})
}

// evaluateFlag resolves the rollout decision for a single flag and user,
// independent of the store. A disabled flag is always false; otherwise the
// deterministic rollout hash decides.
func evaluateFlag(flag store.Flag, user string) bool {
	if !flag.Enabled {
		return false
	}
	return evaluate.Decide(flag.Key, user, flag.RolloutPercent)
}
