package log

import (
	"fmt"

	"github.com/go-logr/logr"
)

// zapLogger is a logr.Logger that uses Zap to log.
// We treat levels 0-3 as info level and 4+ as debug; there are no warnings
// Errors are passed through as errors.
type zapLogger struct {
	l      *Scope
	lvl    int
	lvlSet bool
}

const debugLevelThreshold = 3

func (zl *zapLogger) Enabled() bool {
	if zl.lvlSet && zl.lvl > debugLevelThreshold {
		return zl.l.DebugEnabled()
	}
	return zl.l.InfoEnabled()
}

func (zl *zapLogger) Info(msg string, keysAndVals ...interface{}) {
	if zl.lvlSet && zl.lvl > debugLevelThreshold {
		zl.l.Debug(msg, keysAndVals)
	} else {
		zl.l.Info(msg, keysAndVals)
	}
}

func (zl *zapLogger) Error(err error, msg string, keysAndVals ...interface{}) {
	if zl.l.ErrorEnabled() {
		zl.l.Error(fmt.Sprintf("%v: %s", err.Error(), msg), keysAndVals)
	}
}

func (zl *zapLogger) V(level int) logr.Logger {
	return &zapLogger{
		lvl:    zl.lvl + level,
		l:      zl.l,
		lvlSet: true,
	}
}

func (zl *zapLogger) WithValues(keysAndValues ...interface{}) logr.Logger {
	return newLogrAdapter(zl.l.WithLabels(keysAndValues...))
}

func (zl *zapLogger) WithName(name string) logr.Logger {
	return zl
}

// NewLogger creates a new logr.Logger using the given Zap Logger to log.
func newLogrAdapter(l *Scope) logr.Logger {
	return &zapLogger{
		l:      l,
		lvl:    0,
		lvlSet: false,
	}
}
