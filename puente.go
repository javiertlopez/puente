package puente

import (
	"github.com/sirupsen/logrus"
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
