package logger

// Fields is a convenience alias for structured log fields.
type Fields map[string]interface{}

// Logger defines the minimal interface our app expects from a logger implementation.
// (We adapt logrus to this interface in cmd/main.go.)
type Logger interface {
	Trace(args ...interface{})
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})

	Tracef(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})

	WithFields(fields Fields) Logger
	WithField(key string, value interface{}) Logger
	WithError(err error) Logger
}

var globalLogger Logger

// SetLogger sets the global logger instance used by the app.
func SetLogger(l Logger) { globalLogger = l }

// GetLogger gets the global logger instance (may be nil if not initialized).
func GetLogger() Logger { return globalLogger }
