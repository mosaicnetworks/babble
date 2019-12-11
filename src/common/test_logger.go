package common

import (
	"testing"

	"github.com/sirupsen/logrus"
)

// TestLogLevel is the level used by tests by default.
var TestLogLevel = logrus.DebugLevel

// This can be used as the destination for a logger and it'll
// map them into calls to testing.T.Log, so that you only see
// the logging for failed tests.
type testLoggerAdapter struct {
	t      testing.TB
	prefix string
}

// Write ...
func (a *testLoggerAdapter) Write(d []byte) (int, error) {
	if d[len(d)-1] == '\n' {
		d = d[:len(d)-1]
	}

	// There are 2 blocks of code below: ALTERNATE LOGGING LABEL and STANDARD LOGGING. One block should be
	// commented out using /* */, the other should be uncommented.
	// For the moment, the STANDARD LOGGING should be uncommented in checked in versions.
	// The ALTERNATE LOGGING LABEL code blocks adds a file name and line number to the logging output.
	// The custom logger that we use had the side effect of setting the log location to itself - rather than
	// the calling location.
	// The impact on performance overall is currently undetermined, thus this commented out check in.

	/*
	   //BEGIN ALTERNATE LOGGING LABEL
	   	var skip int
	   // This is hard coded how far up the call stack to traverse
	   // Look at func (c *common) decorate(s string) string  in /usr/lib/go-1.10/src/testing/testing.go
	   // for the original context.

	   	skip = 5

	   	_, file, line, ok := runtime.Caller(skip)
	   	if ok {
	   		// Truncate file name at last file name separator.
	   		if index := strings.LastIndex(file, "/"); index >= 0 {
	   			file = file[index+1:]
	   		} else if index = strings.LastIndex(file, "\\"); index >= 0 {
	   			file = file[index+1:]
	   		}
	   	} else {
	   		file = "???"
	   		line = 1
	   	}

	   	loc := fmt.Sprintf("%s:%d: ", file, line)

	   	if a.prefix != "" {
	   		l := a.prefix + ": " + loc + string(d)
	   		a.t.Log(l)
	   		return len(l), nil
	   	}

	   	l := loc + string(d)
	   	a.t.Log(string(l))
	   	return len(l), nil

	   //END ALTERNATE LOGGING LABEL
	*/
	//BEGIN STANDARD LOGGING

	// Theoretically we could use a call to Testing.TB.Helper() to disregard this function.
	// It would also require some changes to the logger to also pass its functions to be ignored, rendering it
	// non trivial in implementation.

	//	a.t.Helper()

	if a.prefix != "" {
		l := a.prefix + ": " + string(d)
		a.t.Log(l)
		return len(l), nil
	}

	a.t.Log(string(d))
	return len(d), nil
	//END STANDARD LOGGING
}

// NewTestLogger return a logrus Logger for testing
func NewTestLogger(t testing.TB, level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.Out = &testLoggerAdapter{t: t}
	logger.Level = level
	return logger
}

// NewTestEntry returns a logrus Entry for testing
func NewTestEntry(t testing.TB, level logrus.Level) *logrus.Entry {
	logger := NewTestLogger(t, level)
	return logrus.NewEntry(logger)
}
