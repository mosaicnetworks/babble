package common

import (
	"testing"

	"github.com/sirupsen/logrus"
)

// TestLogLevel is the level used in tests by default.
var TestLogLevel = logrus.DebugLevel

// NewTestLogger returns a logrus Logger which displays extra information for
// debugging tests.
func NewTestLogger(t testing.TB, level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetReportCaller(true)
	logger.Level = level
	return logger
}

// NewTestEntry returns a logrus Entry for testing.
func NewTestEntry(t testing.TB, level logrus.Level) *logrus.Entry {
	logger := NewTestLogger(t, level)
	return logrus.NewEntry(logger)
}
