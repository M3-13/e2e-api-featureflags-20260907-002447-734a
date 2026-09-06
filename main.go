package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"featureflagservice/internal/api"
	"featureflagservice/internal/middleware"
	"featureflagservice/internal/store"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	token := os.Getenv("FLAG_API_TOKEN")

	logger := log.Default()

	logger.Printf("featureflagservice listening on %s", addr)
	if err := newServer(addr, token, logger).ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("server error: %v", err)
	}
}

func newServer(addr string, token string, logger *log.Logger) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           newHandler(logger, token),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func newHandler(logger *log.Logger, token string) http.Handler {
	s := store.NewStore()
	handlers := api.NewHandlers(s)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.Healthz)

	mux.Handle("POST /flags", middleware.Auth(token, http.HandlerFunc(handlers.CreateFlag)))
	mux.Handle("GET /flags", middleware.Auth(token, http.HandlerFunc(handlers.ListFlags)))
	mux.Handle("GET /flags/{key}", middleware.Auth(token, http.HandlerFunc(handlers.GetFlag)))
	mux.Handle("PUT /flags/{key}", middleware.Auth(token, http.HandlerFunc(handlers.UpdateFlag)))
	mux.Handle("DELETE /flags/{key}", middleware.Auth(token, http.HandlerFunc(handlers.DeleteFlag)))
	mux.Handle("GET /flags/{key}/evaluate", middleware.Auth(token, http.HandlerFunc(handlers.EvaluateFlag)))

	return middleware.Logging(logger, jsonErrorHandler(mux))
}

// jsonErrorHandler rewrites the plain-text 404/405 responses emitted by the
// ServeMux into the JSON error objects the API contract promises.
func jsonErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ew := &errorWriter{ResponseWriter: w}
		next.ServeHTTP(ew, r)
	})
}

type errorWriter struct {
	http.ResponseWriter
	overridden bool
}

func (w *errorWriter) WriteHeader(code int) {
	// Only rewrite the mux's own plain-text 404/405. A handler that already
	// prepared a JSON response (via writeJSON) sets Content-Type before
	// WriteHeader, so its own 404/405 passes through untouched.
	if (code == http.StatusNotFound || code == http.StatusMethodNotAllowed) &&
		w.Header().Get("Content-Type") != "application/json" {
		w.overridden = true
		msg := "not found"
		if code == http.StatusMethodNotAllowed {
			msg = "method not allowed"
		}
		w.Header().Set("Content-Type", "application/json")
		w.ResponseWriter.WriteHeader(code)
		_ = json.NewEncoder(w.ResponseWriter).Encode(map[string]string{"error": msg})
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *errorWriter) Write(b []byte) (int, error) {
	if w.overridden {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
