package log

import (
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"istio.io/pkg/structured"
)

var (
	toLevel = map[zapcore.Level]Level{
		zapcore.FatalLevel: FatalLevel,
		zapcore.ErrorLevel: ErrorLevel,
		zapcore.WarnLevel:  WarnLevel,
		zapcore.InfoLevel:  InfoLevel,
		zapcore.DebugLevel: DebugLevel,
	}
	toZapLevel = map[Level]zapcore.Level{
		FatalLevel: zapcore.FatalLevel,
		ErrorLevel: zapcore.ErrorLevel,
		WarnLevel:  zapcore.WarnLevel,
		InfoLevel:  zapcore.InfoLevel,
		DebugLevel: zapcore.DebugLevel,
	}
)

func init() {
	RegisterDefaultHandler(ZapLogHandlerCallbackFunc)
}

// ZapLogHandlerCallbackFunc is the handler function that emulates the previous Istio logging output and adds
// support for errdict package and labels logging.
func ZapLogHandlerCallbackFunc(
	level Level,
	scope *Scope,
	ie *structured.Error,
	msg string,
	fields []zapcore.Field) {
	if ie != nil {
		fields = append(fields, zap.String("message", msg))
		fields = append(fields, zap.String("moreInfo", ie.MoreInfo))
		fields = append(fields, zap.String("impact", ie.Impact))
		fields = append(fields, zap.String("action", ie.Action))
		fields = append(fields, zap.String("likelyCauses", ie.LikelyCause))
	}
	if len(scope.labelKeys) > 0 {
		for _, k := range scope.labelKeys {
			v := scope.labels[k]
			fields = append(fields, zap.Field{
				Key:       k,
				Type:      zapcore.ReflectType,
				Interface: v,
			})
		}
	}
	emit(scope, toZapLevel[level], msg, fields)
}

func toZapSlice(index int, fields ...interface{}) []zapcore.Field {
	var zfs []zapcore.Field
	if len(fields) <= index {
		return nil
	}
	for _, zfi := range fields {
		zf, ok := zfi.(zapcore.Field)
		if !ok {
			Errorf("bad interface type: expect zapcore.Field, got %T for fields %v", zf, fields)
			continue
		}
		zfs = append(zfs, zf)
	}
	return zfs
}

// callerSkipOffset is how many callers to pop off the stack to determine the caller function locality, used for
// adding file/line number to log output.
const callerSkipOffset = 4

func dumpStack(level zapcore.Level, scope *Scope) bool {
	thresh := toLevel[level]
	if scope != defaultScope {
		thresh = ErrorLevel
		switch level {
		case zapcore.FatalLevel:
			thresh = FatalLevel
		}
	}
	return scope.GetStackTraceLevel() >= thresh
}

func emit(scope *Scope, level zapcore.Level, msg string, fields []zapcore.Field) {
	e := zapcore.Entry{
		Message:    msg,
		Level:      level,
		Time:       time.Now(),
		LoggerName: scope.nameToEmit,
	}

	if scope.GetLogCallers() {
		e.Caller = zapcore.NewEntryCaller(runtime.Caller(scope.callerSkip + callerSkipOffset))
	}

	if dumpStack(level, scope) {
		e.Stack = zap.Stack("").String
	}

	pt := funcs.Load().(patchTable)
	if pt.write != nil {
		if err := pt.write(e, fields); err != nil {
			_, _ = fmt.Fprintf(pt.errorSink, "%v log write error: %v\n", time.Now(), err)
			_ = pt.errorSink.Sync()
		}
	}
}

// backwards compatibility. TODO(mostrowski): remove this
func (s *Scope) emit(level zapcore.Level, ps bool, msg string, fields []zapcore.Field) {
	emit(s, level, msg, fields)
}
