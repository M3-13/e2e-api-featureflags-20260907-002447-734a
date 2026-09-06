package api

import "net/http"

func (h *Handlers) EvaluateFlag(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
