package logger

import (
	"fmt"
	"log"

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
	return newLogger(4, "panacea-ingress-controller", nil).Logger
}

func NewLoggerWithLevel(lvl int) PanaceaLogger {
	return newLogger(lvl, "panacea-ingress-controller", nil)
}

func newLogger(logLevel int, name string, kv []any) PanaceaLogger {

	_logger := funcr.New(func(prefix, args string) {
		if prefix != "" {
			fmt.Printf("%s %s\n", prefix, args)
		} else {
			fmt.Println(args)
		}
	}, funcr.Options{
		LogTimestamp:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		Verbosity:       logLevel,
		LogInfoLevel:    getLogLevel(logLevel),
		MaxLogDepth:     8,
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
		*level = "DEBUG"
	case 2:
		*level = "TRACE"
	default:
		*level = "INFO"
	}
	return level
}
