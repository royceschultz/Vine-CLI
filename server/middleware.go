package server

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestLogger returns middleware that logs each request.
func RequestLogger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			logger.Printf("%s %s %d %s %s", r.Method, r.URL.Path, rw.status, time.Since(start).Round(time.Microsecond), r.RemoteAddr)
		})
	}
}

// TokenAuth returns middleware that checks for a valid Bearer token.
// If token is empty, the middleware is a no-op (localhost-only mode).
func TokenAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if token == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Health endpoint is always accessible (used for server discovery).
			if r.URL.Path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != token {
				writeError(w, http.StatusUnauthorized, "invalid or missing token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
