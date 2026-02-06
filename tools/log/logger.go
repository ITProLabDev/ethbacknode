// Package log provides structured logging utilities for the application.
// It wraps the go-logging library with convenient helper functions.
package log

import (
	"encoding/json"
	"os"
	"reflect"

	"github.com/DeepForestTeam/go-logging"
)

// DEFAULT_DEBUG_LEVEL is the default logging verbosity level.
const (
	DEFAULT_DEBUG_LEVEL = 5
)

// Package-level logging state.
var (
	debugMode int
	_loger    = logging.MustGetLogger("main")
	format    = make(map[int]logging.Formatter)

	backend *logging.LogBackend
)

// GetLogger returns the underlying go-logging logger instance.
func GetLogger() *logging.Logger {
	return _loger
}

func init() {
	format[1] = logging.MustStringFormatter(`%{color}%{time:15:04:05} [%{level:.5s}] %{longfile}` + "\t▶ %{longpkg}::%{longfunc}::%{callpath} \t" + `| %{message}%{color:reset}`)
	format[2] = logging.MustStringFormatter(`%{color}%{time:15:04:05} [%{level:.5s}] %{longfile}` + "\t▶ %{longpkg}::%{longfunc} \t" + `| %{message}%{color:reset}`)
	format[3] = logging.MustStringFormatter(`%{color}%{time:15:04:05} [%{level:.5s}] %{shortfile}` + "\t▶ %{longfunc} \t" + `| %{message}%{color:reset}`)
	format[4] = logging.MustStringFormatter(`%{color}%{time:15:04:05} [%{level:.4s}] %{shortfile}` + "\t %{shortfunc} \t" + `| %{message}%{color:reset}`)
	format[5] = logging.MustStringFormatter(`%{color}%{time:15:04:05} [%{level:.4s}] %{shortfile}` + "\t" + `| %{message}%{color:reset}`)
	format[6] = logging.MustStringFormatter(`%{color}%{time:15:04:05} [%{level:.3s}]` + "\t" + `| %{message}%{color:reset}`)
	format[7] = logging.MustStringFormatter(`%{color}%{time:15:04}` + "\t" + `| %{message}%{color:reset}`)
	debugMode = DEFAULT_DEBUG_LEVEL
	backend = logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format[debugMode])
	logging.SetBackend(backendFormatter)
	_loger.ExtraCalldepth = 1
}

// SetLevel changes the logging verbosity level (1-7).
func SetLevel(l int) {
	debugMode = l
	backend = logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format[debugMode])
	logging.SetBackend(backendFormatter)
	_loger.ExtraCalldepth = 1
}

// Println logs a debug message.
func Println(v ...interface{}) {
	_loger.Debug(v...)
}

// Dump logs values as formatted JSON with type information.
func Dump(v ...interface{}) {
	if len(v) == 1 {
		val := v[0]
		if val == nil {
			_loger.Debug("NIL OBJECT")
			return
		}
		typeName := reflect.TypeOf(val).String()
		b, _ := json.MarshalIndent(val, "", "\t")
		_loger.Debug("\nTYPE: [", typeName, "]:\n", string(b))
	} else {
		for i, val := range v {
			isString := reflect.TypeOf(val) == reflect.TypeOf("")
			if !isString {
				if val == nil {
					_loger.Debug("NIL OBJECT")
					continue
				}
				typeName := reflect.TypeOf(val).String()
				b, _ := json.MarshalIndent(val, "", "\t")
				_loger.Debug("\nTYPE: [", typeName, "]:\n", string(b))
				if i < len(v)-1 {
					_loger.Debug("\n\n")
				}
			} else {
				_loger.Debug("Dump:", val)
			}
		}
	}
}

// Debug logs a debug-level message.
func Debug(v ...interface{}) {
	_loger.Debug(v...)
}
// Info logs an info-level message.
func Info(v ...interface{}) {
	_loger.Info(v...)
}
// Fatal logs a fatal-level message and exits the application.
func Fatal(v ...interface{}) {
	_loger.Fatal(v...)
}
// Panic logs a message and panics.
func Panic(v ...interface{}) {
	_loger.Panic(v...)
}
// Critical logs a critical-level message.
func Critical(v ...interface{}) {
	_loger.Critical(v...)
}
// Error logs an error-level message.
func Error(v ...interface{}) {
	_loger.Error(v...)
}
// Warning logs a warning-level message.
func Warning(v ...interface{}) {
	_loger.Warning(v...)
}
// Notice logs a notice-level message.
func Notice(v ...interface{}) {
	_loger.Notice(v...)
}
