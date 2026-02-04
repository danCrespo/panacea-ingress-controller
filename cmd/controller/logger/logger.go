package logger

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
)

type PanaceaLogger struct {
	Logger        logr.Logger
	keysAndValues []any
	name          string
	logLevel      int
}

var (
	Log = NewLogger()
)

func NewLogger() logr.Logger {
	return newLogger(4, "panacea-controller", nil).Logger
}

func NewLoggerWithLevel(lvl int) logr.Logger {
	return newLogger(lvl, "panacea-controller", nil).Logger
}

func replaceAll(s string, oldNewPairs ...string) string {
	if len(oldNewPairs)%2 != 0 {
		return s
	}
	for i := 0; i < len(oldNewPairs); i += 2 {
		s = strings.ReplaceAll(s, oldNewPairs[i], oldNewPairs[i+1])
	}
	return s
}

func newLogger(logLevel int, name string, kv []any) PanaceaLogger {

	_logger := funcr.New(func(prefix, args string) {
		if prefix == "" {
			msg := replaceAll(args, "\\n", "\n")
			fmt.Println(msg)
			return
		}
		msg := replaceAll(prefix, "%s", args, "%d", args, "%v", args)
		msg = replaceAll(msg, "\\n", "\n")
		fmt.Println(msg)

	}, funcr.Options{
		LogTimestamp:    logLevel > 0,
		TimestampFormat: "02-01-06 15:04:05",
		Verbosity:       logLevel,
		LogInfoLevel:    getLogLevel(logLevel),
		MaxLogDepth:     logLevel + 1,
		LogCaller:       funcr.MessageClass(logLevel),
		LogCallerFunc:   true,
	})
	return PanaceaLogger{
		Logger:        _logger,
		name:          name,
		logLevel:      logLevel,
		keysAndValues: kv,
	}
}

func (l *PanaceaLogger) Fatal(v ...any) {
	log.Fatal(v...)
}

func (l *PanaceaLogger) Println(v ...any) {
	log.Println(v...)
}

func (l *PanaceaLogger) Enabled() bool {
	return l.Logger.Enabled()
}

func (l *PanaceaLogger) GetSink() logr.LogSink {
	return l.Logger.GetSink()
}

func (l *PanaceaLogger) GetV() int {
	return l.logLevel
}

func (l *PanaceaLogger) IsZero() bool {
	return l.Logger.IsZero()
}

func (l *PanaceaLogger) WithCallDepth(depth int) logr.Logger {
	return l.Logger.WithCallDepth(depth)
}

func (l *PanaceaLogger) WithCallStackHelper() (func(), logr.Logger) {
	return l.Logger.WithCallStackHelper()
}

func (l *PanaceaLogger) Info(msg string, keysAndValues ...any) {
	l.Logger.Info(msg, keysAndValues...)
}

func (l *PanaceaLogger) Error(err error, msg string, keysAndValues ...any) {
	l.Logger.Error(err, msg, keysAndValues...)
}

func (l *PanaceaLogger) WithValues(keysAndValues ...any) logr.Logger {
	return l.Logger.WithValues(keysAndValues...)
}

func (l *PanaceaLogger) WithName(name string) logr.Logger {
	return l.Logger.WithName(name)
}

func (l *PanaceaLogger) V(level int) logr.Logger {
	return l.Logger.V(level)
}

func getLogLevel(logLevel int) (level *string) {
	level = new(string)
	switch logLevel {
	case 0:
		*level = "INFO"
	case 1:
		*level = "VERBOSE"
	case 2:
		*level = "DEBUG"
	case 3:
		*level = "TRACE"
	default:
		*level = "INFO"
	}
	return level
}
