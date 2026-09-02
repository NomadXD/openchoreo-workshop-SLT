package main

import "net/http"

// withCORS allows employee-console-ui to call this service directly from
// the browser (it's not routed through any BFF/proxy — see the top-level
// demo README). Wide open (any origin, no credentials) is an acceptable
// tradeoff here: neither this service nor its caller uses cookies, and it
// has no auth mechanism of its own regardless of origin.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Actor-Role, X-Actor-Id")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
