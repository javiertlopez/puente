package puente

import (
	"context"
	"net/http"
	"time"
)

// responseWriter wraps the http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// newResponseWriter returns a responseWriter wrapper to access the http status
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader keeps the status code
func (r *responseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) (string, bool) {
	v := ctx.Value(RequestIDKey)
	if v == nil {
		return "", false
	}

	requestID, ok := v.(string)
	return requestID, ok
}

// Logging middleware logs the request
func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get or generate a request ID
			requestID := r.Context().Value(RequestIDKey)
			if requestID == nil {
				requestID = generateRequestID()
				// Add request ID to context
				ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
				r = r.WithContext(ctx)
			}

			// Use the wrapped response writer to capture status code
			wrapped := newResponseWriter(w)
			next.ServeHTTP(wrapped, r)

			// Prepare log fields with defaults
			logFields := m.defaultLogFields()
			logFields["request_id"] = requestID
			logFields["status"] = wrapped.statusCode
			logFields["method"] = r.Method
			logFields["path"] = r.URL.EscapedPath()
			logFields["duration"] = time.Since(start)

			// Add user ID if available
			userId, ok := GetUserID(r.Context())
			if !ok {
				m.logger.WithFields(logFields).Warn("Failed to get user ID from context")
			}

			logFields["user_id"] = userId
			m.logger.WithFields(logFields).Info("Request completed")
		},
	)
}
