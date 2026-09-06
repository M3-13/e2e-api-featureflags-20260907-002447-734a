package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Auth guards the wrapped handler behind a Bearer-token check. It fails closed:
// an empty token (FLAG_API_TOKEN unset), a missing Authorization header, or a
// header that does not exactly match "Bearer <token>" all answer 401 with the
// JSON error object {"error":"unauthorized"}. The token itself is never logged
// or echoed back in a response.
func Auth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			unauthorized(w)
			return
		}
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, prefix) || strings.TrimPrefix(auth, prefix) != token {
			unauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
