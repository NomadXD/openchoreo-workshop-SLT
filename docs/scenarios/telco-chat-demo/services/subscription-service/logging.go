package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the status code written.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// withLogging logs one structured line per request: method, path, status,
// actorRole, actorId, latency. X-Actor-Role and X-Actor-Id are optional
// headers read purely for logging; no authorization logic is applied.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		entry := map[string]interface{}{
			"method":    r.Method,
			"path":      r.URL.Path,
			"status":    rec.status,
			"actorRole": r.Header.Get("X-Actor-Role"),
			"actorId":   r.Header.Get("X-Actor-Id"),
			"latencyMs": time.Since(start).Milliseconds(),
		}
		line, err := json.Marshal(entry)
		if err != nil {
			log.Printf("method=%s path=%s status=%d", r.Method, r.URL.Path, rec.status)
			return
		}
		log.Println(string(line))
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
