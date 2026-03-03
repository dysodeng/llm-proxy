package middleware

import (
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// responseWriter wraps http.ResponseWriter to capture the status code and
// the number of bytes written in the response body.
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

// newResponseWriter returns a responseWriter that defaults to status 200.
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

// WriteHeader captures the status code and delegates to the underlying writer.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the number of bytes written and delegates to the underlying writer.
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// Flush implements http.Flusher to support SSE streaming.
// It delegates to the underlying ResponseWriter if it also implements http.Flusher.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// maskAPIKey returns the last 4 characters of the key, prefixed with "****".
// If key is shorter than 4 chars, returns "****".
func maskAPIKey(key string) string {
	if len(key) < 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

// extractProvider returns "openai" or "anthropic" based on the URL path prefix.
// Returns "unknown" for unrecognized paths.
func extractProvider(path string) string {
	switch {
	case strings.HasPrefix(path, "/openai"):
		return "openai"
	case strings.HasPrefix(path, "/anthropic"):
		return "anthropic"
	default:
		return "unknown"
	}
}

// Logging returns an HTTP middleware that logs each request using zap.
// Log fields: provider, path, api_key (masked last 4), status, latency_ms, req_bytes, resp_bytes.
func Logging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Resolve api key: prefer x-api-key, fall back to Authorization.
			apiKey := r.Header.Get("x-api-key")
			if apiKey == "" {
				auth := r.Header.Get("Authorization")
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}

			// Determine request body size; use 0 when unknown (-1).
			var reqBytes int64
			if r.ContentLength > 0 {
				reqBytes = r.ContentLength
			}

			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)

			latencyMs := time.Since(start).Milliseconds()
			provider := extractProvider(r.URL.Path)

			logger.Info("request",
				zap.String("provider", provider),
				zap.String("path", r.URL.Path),
				zap.String("api_key", maskAPIKey(apiKey)),
				zap.Int("status", rw.status),
				zap.Int64("latency_ms", latencyMs),
				zap.Int64("req_bytes", reqBytes),
				zap.Int("resp_bytes", rw.bytes),
			)
		})
	}
}
