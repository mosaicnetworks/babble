package common

import (
	"testing"

	"github.com/sirupsen/logrus"
)

// TestLogLevel is the level used in tests by default.
var TestLogLevel = logrus.InfoLevel

// This can be used as the destination for a logger and it will map them into
// calls to testing.T.Log, so that you only see the logging for failed tests.
type testLoggerAdapter struct {
	t      testing.TB
	prefix string
}

// Write implements io.Writer interface.
func (a *testLoggerAdapter) Write(d []byte) (int, error) {
	if d[len(d)-1] == '\n' {
		d = d[:len(d)-1]
	}

	if a.prefix != "" {
		l := a.prefix + ": " + string(d)
		a.t.Log(l)
		return len(l), nil
	}

	a.t.Log(string(d))
	return len(d), nil
}

// NewTestLogger return a logrus Logger for testing
func NewTestLogger(t testing.TB, level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.Out = &testLoggerAdapter{t: t}
	// logger.SetReportCaller(true)
	logger.Level = level
	return logger
}

// NewTestEntry returns a logrus Entry for testing.
func NewTestEntry(t testing.TB, level logrus.Level) *logrus.Entry {
	logger := NewTestLogger(t, level)
	return logrus.NewEntry(logger)
}
