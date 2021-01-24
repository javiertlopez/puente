package puente

import "github.com/sirupsen/logrus"

// Middleware holds the app name and logger
type Middleware struct {
	app    string
	logger *logrus.Logger
}

// New returns a middleware instance
func New(app string, logger *logrus.Logger) *Middleware {
	return &Middleware{
		app:    app,
		logger: logger,
	}
}
