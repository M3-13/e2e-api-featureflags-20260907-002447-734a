package api

import (
	"net/http"
	"regexp"

	"featureflagservice/internal/store"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func validKey(k string) bool {
	return keyPattern.MatchString(k)
}

func (h *Handlers) CreateFlag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key            string `json:"key"`
		Enabled        *bool  `json:"enabled"`
		Description    string `json:"description"`
		RolloutPercent *int   `json:"rollout_percent"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	if body.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if !validKey(body.Key) {
		writeError(w, http.StatusBadRequest, "key may only contain [A-Za-z0-9._-]")
		return
	}
	if body.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}
	rollout := 0
	if body.RolloutPercent != nil {
		rollout = *body.RolloutPercent
	}
	if rollout < 0 || rollout > 100 {
		writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}
	flag := store.Flag{
		Key:            body.Key,
		Enabled:        *body.Enabled,
		Description:    body.Description,
		RolloutPercent: rollout,
	}
	if err := h.store.Create(flag); err != nil {
		writeError(w, http.StatusConflict, "flag already exists")
		return
	}
	writeJSON(w, http.StatusCreated, flag)
}

func (h *Handlers) ListFlags(w http.ResponseWriter, r *http.Request) {
	flags := h.store.GetAll()
	if flags == nil {
		flags = []store.Flag{}
	}
	writeJSON(w, http.StatusOK, flags)
}

func (h *Handlers) GetFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	flag, ok := h.store.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (h *Handlers) UpdateFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body struct {
		Enabled        *bool  `json:"enabled"`
		Description    string `json:"description"`
		RolloutPercent *int   `json:"rollout_percent"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}
	if body.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}
	rollout := 0
	if body.RolloutPercent != nil {
		rollout = *body.RolloutPercent
	}
	if rollout < 0 || rollout > 100 {
		writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}
	update := store.FlagUpdate{
		Enabled:        *body.Enabled,
		Description:    body.Description,
		RolloutPercent: rollout,
	}
	flag, err := h.store.Update(key, update)
	if err != nil {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (h *Handlers) DeleteFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if h.store.Delete(key) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeError(w, http.StatusNotFound, "flag not found")
}
