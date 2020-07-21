// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"go.uber.org/zap/zapcore"

	"istio.io/pkg/structured"
)

// Scope let's you log data for an area of code, enabling the user full control over
// the level of logging output produced.
type Scope struct {
	// immutable, set at creation
	name        string
	nameToEmit  string
	description string
	callerSkip  int

	// set by the Configure method and adjustable dynamically
	outputLevel     atomic.Value
	stackTraceLevel atomic.Value
	logCallers      atomic.Value

	// labels data - key slice to preserve ordering
	labelKeys []string
	labels    map[string]interface{}
}

var (
	scopes          = make(map[string]*Scope)
	defaultHandlers []scopeHandlerCallbackFunc
	lock            sync.RWMutex
)

// scopeHandlerCallbackFunc is a callback type for the handler called from Fatal*, Error*, Warn*, Info* and Debug*
// function calls.
type scopeHandlerCallbackFunc func(
	level Level,
	scope *Scope,
	ie *structured.Error,
	msg string,
	fields []zapcore.Field)

// registerDefaultHandler registers a scope handler that is called by default from all scopes. It is appended to the
// current list of scope handlers.
func registerDefaultHandler(callback scopeHandlerCallbackFunc) {
	lock.Lock()
	defer lock.Unlock()
	defaultHandlers = append(defaultHandlers, callback)
}

// RegisterScope registers a new logging scope. If the same name is used multiple times
// for a single process, the same Scope struct is returned.
//
// Scope names cannot include colons, commas, or periods.
func RegisterScope(name string, description string, callerSkip int) *Scope {
	if strings.ContainsAny(name, ":,.") {
		panic(fmt.Sprintf("scope name %s is invalid, it cannot contain colons, commas, or periods", name))
	}

	lock.Lock()
	defer lock.Unlock()

	s, ok := scopes[name]
	if !ok {
		s = &Scope{
			name:        name,
			description: description,
			callerSkip:  callerSkip,
		}
		s.SetOutputLevel(InfoLevel)
		s.SetStackTraceLevel(NoneLevel)
		s.SetLogCallers(false)

		if name != DefaultScopeName {
			s.nameToEmit = name
		}

		scopes[name] = s
	}

	s.labels = make(map[string]interface{})

	return s
}

// FindScope returns a previously registered scope, or nil if the named scope wasn't previously registered
func FindScope(scope string) *Scope {
	lock.RLock()
	defer lock.RUnlock()

	s := scopes[scope]
	return s
}

// Scopes returns a snapshot of the currently defined set of scopes
func Scopes() map[string]*Scope {
	lock.RLock()
	defer lock.RUnlock()

	s := make(map[string]*Scope, len(scopes))
	for k, v := range scopes {
		s[k] = v
	}

	return s
}

// Fatal outputs a message at fatal level.
func (s *Scope) Fatal(fields ...interface{}) {
	if s.GetOutputLevel() >= FatalLevel {
		ie, firstIdx := getErrorStruct(fields)
		s.callHandlers(FatalLevel, s, ie, fields[0].(string), toZapSlice(firstIdx+1, fields))
	}
}

// Fatala uses fmt.Sprint to construct and log a message at fatal level.
func (s *Scope) Fatala(args ...interface{}) {
	if s.GetOutputLevel() >= FatalLevel {
		ie, firstIdx := getErrorStruct(args)
		s.callHandlers(FatalLevel, s, ie, fmt.Sprint(args[firstIdx:]...), nil)
	}
}

// Fatalf uses fmt.Sprintf to construct and log a message at fatal level.
func (s *Scope) Fatalf(args ...interface{}) {
	if s.GetOutputLevel() >= FatalLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(FatalLevel, s, ie, msg, nil)
	}
}

// FatalEnabled returns whether output of messages using this scope is currently enabled for fatal-level output.
func (s *Scope) FatalEnabled() bool {
	return s.GetOutputLevel() >= FatalLevel
}

// Error outputs a message at error level.
func (s *Scope) Error(fields ...interface{}) {
	if s.GetOutputLevel() >= ErrorLevel {
		ie, firstIdx := getErrorStruct(fields)
		s.callHandlers(ErrorLevel, s, ie, fields[0].(string), toZapSlice(firstIdx+1, fields))
	}
}

// Errora uses fmt.Sprint to construct and log a message at error level.
func (s *Scope) Errora(args ...interface{}) {
	if s.GetOutputLevel() >= ErrorLevel {
		ie, firstIdx := getErrorStruct(args)
		s.callHandlers(ErrorLevel, s, ie, fmt.Sprint(args[firstIdx:]...), nil)
	}
}

// Errorf uses fmt.Sprintf to construct and log a message at error level.
func (s *Scope) Errorf(args ...interface{}) {
	if s.GetOutputLevel() >= ErrorLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(ErrorLevel, s, ie, msg, nil)
	}
}

// ErrorEnabled returns whether output of messages using this scope is currently enabled for error-level output.
func (s *Scope) ErrorEnabled() bool {
	return s.GetOutputLevel() >= ErrorLevel
}

// Warn outputs a message at warn level.
func (s *Scope) Warn(fields ...interface{}) {
	if s.GetOutputLevel() >= WarnLevel {
		ie, firstIdx := getErrorStruct(fields)
		s.callHandlers(WarnLevel, s, ie, fields[0].(string), toZapSlice(firstIdx+1, fields))
	}
}

// Warna uses fmt.Sprint to construct and log a message at warn level.
func (s *Scope) Warna(args ...interface{}) {
	if s.GetOutputLevel() >= WarnLevel {
		ie, firstIdx := getErrorStruct(args)
		s.callHandlers(WarnLevel, s, ie, fmt.Sprint(args[firstIdx:]...), nil)
	}
}

// Warnf uses fmt.Sprintf to construct and log a message at warn level.
func (s *Scope) Warnf(args ...interface{}) {
	if s.GetOutputLevel() >= WarnLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(WarnLevel, s, ie, msg, nil)
	}
}

// WarnEnabled returns whether output of messages using this scope is currently enabled for warn-level output.
func (s *Scope) WarnEnabled() bool {
	return s.GetOutputLevel() >= WarnLevel
}

// Info outputs a message at info level.
func (s *Scope) Info(fields ...interface{}) {
	if s.GetOutputLevel() >= InfoLevel {
		ie, firstIdx := getErrorStruct(fields)
		s.callHandlers(InfoLevel, s, ie, fields[0].(string), toZapSlice(firstIdx+1, fields))
	}
}

// Infoa uses fmt.Sprint to construct and log a message at info level.
func (s *Scope) Infoa(args ...interface{}) {
	if s.GetOutputLevel() >= InfoLevel {
		ie, firstIdx := getErrorStruct(args)
		s.callHandlers(InfoLevel, s, ie, fmt.Sprint(args[firstIdx:]...), nil)
	}
}

// Infof uses fmt.Sprintf to construct and log a message at info level.
func (s *Scope) Infof(args ...interface{}) {
	if s.GetOutputLevel() >= InfoLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(InfoLevel, s, ie, msg, nil)
	}
}

// InfoEnabled returns whether output of messages using this scope is currently enabled for info-level output.
func (s *Scope) InfoEnabled() bool {
	return s.GetOutputLevel() >= InfoLevel
}

// Debug outputs a message at debug level.
func (s *Scope) Debug(fields ...interface{}) {
	if s.GetOutputLevel() >= DebugLevel {
		ie, firstIdx := getErrorStruct(fields)
		s.callHandlers(DebugLevel, s, ie, fields[0].(string), toZapSlice(firstIdx+1, fields))
	}
}

// Debuga uses fmt.Sprint to construct and log a message at debug level.
func (s *Scope) Debuga(args ...interface{}) {
	if s.GetOutputLevel() >= DebugLevel {
		ie, firstIdx := getErrorStruct(args)
		s.callHandlers(DebugLevel, s, ie, fmt.Sprint(args[firstIdx:]...), nil)
	}
}

// Debugf uses fmt.Sprintf to construct and log a message at debug level.
func (s *Scope) Debugf(args ...interface{}) {
	if s.GetOutputLevel() >= DebugLevel {
		ie, firstIdx := getErrorStruct(args)
		msg := fmt.Sprint(args[firstIdx])
		if len(args) > 1 {
			msg = fmt.Sprintf(msg, args[firstIdx+1:]...)
		}
		s.callHandlers(DebugLevel, s, ie, msg, nil)
	}
}

// DebugEnabled returns whether output of messages using this scope is currently enabled for debug-level output.
func (s *Scope) DebugEnabled() bool {
	return s.GetOutputLevel() >= DebugLevel
}

// Name returns this scope's name.
func (s *Scope) Name() string {
	return s.name
}

// Description returns this scope's description
func (s *Scope) Description() string {
	return s.description
}

// SetOutputLevel adjusts the output level associated with the scope.
func (s *Scope) SetOutputLevel(l Level) {
	s.outputLevel.Store(l)
}

// GetOutputLevel returns the output level associated with the scope.
func (s *Scope) GetOutputLevel() Level {
	return s.outputLevel.Load().(Level)
}

// SetStackTraceLevel adjusts the stack tracing level associated with the scope.
func (s *Scope) SetStackTraceLevel(l Level) {
	s.stackTraceLevel.Store(l)
}

// GetStackTraceLevel returns the stack tracing level associated with the scope.
func (s *Scope) GetStackTraceLevel() Level {
	return s.stackTraceLevel.Load().(Level)
}

// SetLogCallers adjusts the output level associated with the scope.
func (s *Scope) SetLogCallers(logCallers bool) {
	s.logCallers.Store(logCallers)
}

// GetLogCallers returns the output level associated with the scope.
func (s *Scope) GetLogCallers() bool {
	return s.logCallers.Load().(bool)
}

// Copy makes a copy of s and returns a pointer to it.
func (s *Scope) Copy() *Scope {
	out := *s
	out.labels = copyStringInterfaceMap(s.labels)
	return &out
}

// WithLabels adds a key-value pairs to the labels in s.
func (s *Scope) WithLabels(kvlist ...interface{}) *Scope {
	out := s.Copy()
	if len(kvlist)%2 != 0 {
		out.labels["WithLabels error"] = fmt.Sprintf("even number of parameters required, got %d", len(kvlist))
		return out
	}

	for i := 0; i < len(kvlist); i += 2 {
		keyi := kvlist[i]
		key, ok := keyi.(string)
		if !ok {
			out.labels["WithLabels error"] = fmt.Sprintf("label name %v must be a string, got %T ", keyi, keyi)
			return out
		}
		out.labels[key] = kvlist[i+1]
		out.labelKeys = append(out.labelKeys, key)
	}
	return out
}

// WithoutLabels makes a copy of s, clears labels in s with the given keys and returns the copy.
// Not-existent keys are ignored.
func (s *Scope) WithoutLabels(keys ...string) *Scope {
	out := s.Copy()
	for _, key := range keys {
		delete(out.labels, key)
		out.labelKeys = removeKey(out.labelKeys, key)
	}
	return out
}

// WithoutAnyLabels clears all labels from a copy of s and returns it.
func (s *Scope) WithoutAnyLabels() *Scope {
	out := s.Copy()
	out.labels = make(map[string]interface{})
	out.labelKeys = nil
	return out
}

// callHandlers calls all handlers registered to s.
func (s *Scope) callHandlers(
	severity Level,
	scope *Scope,
	ie *structured.Error,
	msg string,
	fields []zapcore.Field) {
	for _, h := range defaultHandlers {
		h(severity, scope, ie, msg, fields)
	}
}

// getErrorStruct returns (*Error, 1) if it is the first argument in the list is an Error ptr,
// or (nil,0) otherwise. The second return value is the offset to the first non-Error field.
func getErrorStruct(fields ...interface{}) (*structured.Error, int) {
	ief, ok := fields[0].([]interface{})
	if !ok {
		return nil, 0
	}
	ie, ok := ief[0].(*structured.Error)
	if !ok {
		return nil, 0
	}
	// Skip Error, pass remaining fields on as before.
	return ie, 1
}

func copyStringInterfaceMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func removeKey(s []string, key string) []string {
	for i, k := range s {
		if k == key {
			s = append(s[:i], s[i+1:]...)
			return s
		}
	}
	return s
}
