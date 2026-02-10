package middleware

import (
	"net/http"
	"strings"
)

func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, o := range origins {
		allowed[strings.TrimSpace(o)] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			// Allow exact match from config
			if allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if origin != "" && (strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")) {
				// Allow any localhost / 127.0.0.1 port in development
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(origins) > 0 {
				w.Header().Set("Access-Control-Allow-Origin", origins[0])
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
