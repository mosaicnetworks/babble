package mobile

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger
	lock   sync.Once
)

// Logger initialize single logger object
func Logger() *logrus.Logger {
	lock.Do(func() {
		logger = logrus.New()
		logger.Formatter = new(logrus.JSONFormatter)
		logger.Level = logrus.DebugLevel
	})
	return logger
}
