package puente

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// ResponseWriter returns a responseWritter wrapper to access the http status
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader keeps the status code
func (r *responseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Logging middleware logs the request
func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := newResponseWriter(w)
			next.ServeHTTP(wrapped, r)

			m.logger.WithFields(log.Fields{
				"app":      m.app,
				"status":   wrapped.statusCode,
				"method":   r.Method,
				"path":     r.URL.EscapedPath(),
				"duration": time.Since(start),
				"user_id":  r.Context().Value(userIDKey),
			}).Info()
		},
	)
}
