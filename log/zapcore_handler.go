package log

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/prometheus/common/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"istio.io/pkg/errdict"
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

func ZapLogHandlerCallbackFunc(
	level Level,
	scope *Scope,
	ie *errdict.IstioErrorStruct,
	msg string,
	fields []zapcore.Field) {
	var ss []string
	if ie != nil {
		ss = append(ss, fmt.Sprintf(`"moreInfo":"%s"`, ie.MoreInfo))
		ss = append(ss, fmt.Sprintf(`"impact":"%s"`, ie.Impact))
		ss = append(ss, fmt.Sprintf(`"action":"%s"`, ie.Action))
		ss = append(ss, fmt.Sprintf(`"likelyCauses":"%s"`, ie.LikelyCauses))
	}
	if len(scope.structuredKey) > 0 {
		scope.mu.RLock()
		for _, k := range scope.structuredKey {
			v := scope.structured[k]
			if _, ok := v.(string); ok {
				ss = append(ss, fmt.Sprintf(`"%s":"%v"`, k, v))
			} else {
				ss = append(ss, fmt.Sprintf(`"%s":%v`, k, v))
			}
		}
		scope.mu.RUnlock()
	}
	if len(ss) > 0 {
		ss = append(ss, fmt.Sprintf(`"message":"%s"`, msg))
		msg = "{" + strings.Join(ss, ",") + "}"
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
			log.Errorf("bad interface type: expect zapcore.Field, got %T for fields %v", zf, fields)
			continue
		}
		zfs = append(zfs, zf)
	}
	return zfs
}

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
