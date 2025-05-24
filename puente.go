package puente

import (
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type contextKey string

const (
	// UserIDKey is the key used to store the user ID in context
	UserIDKey contextKey = "user_id"
	// RequestIDKey is the key used to store the request ID in context
	RequestIDKey contextKey = "request_id"
)

// Middleware holds the app name and logger
type Middleware struct {
	app       string
	logger    *logrus.Logger
	extractor JWTExtractor
}

// New creates a new Middleware instance
func New(app string, logger *logrus.Logger, extractor JWTExtractor) *Middleware {
	return &Middleware{
		app:       app,
		logger:    logger,
		extractor: extractor,
	}
}

// generateRequestID creates a unique request ID
func generateRequestID() string {
	return uuid.New().String()
}

// defaultLogFields returns the default log fields for any log entry
func (m *Middleware) defaultLogFields() logrus.Fields {
	return logrus.Fields{
		"app":       m.app,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
}
