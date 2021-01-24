package puente

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// Logging middleware logs the request
func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			log.WithFields(log.Fields{
				"status":         r.Response.Status,
				"method":         r.Method,
				"path":           r.URL.EscapedPath(),
				"duration":       time.Since(start),
				"content-length": r.ContentLength,
				"x-origin-id":    r.Header.Get("x-origin-id"),
			}).Info()
		},
	)
}
