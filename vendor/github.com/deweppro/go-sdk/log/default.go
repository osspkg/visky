package log

import "io"

var std = New()

// Default logger
func Default() Logger {
	return std
}

// SetOutput change writer
func SetOutput(out io.Writer) {
	std.SetOutput(out)
}

// SetLevel change log level
func SetLevel(v uint32) {
	std.SetLevel(v)
}

// GetLevel getting log level
func GetLevel() uint32 {
	return std.GetLevel()
}

// Close waiting for all messages to finish recording
func Close() {
	std.Close()
}

// Infof info message
func Infof(format string, args ...interface{}) {
	std.Infof(format, args...)
}

// Warnf warning message
func Warnf(format string, args ...interface{}) {
	std.Warnf(format, args...)
}

// Errorf error message
func Errorf(format string, args ...interface{}) {
	std.Errorf(format, args...)
}

// Debugf debug message
func Debugf(format string, args ...interface{}) {
	std.Debugf(format, args...)
}

// Fatalf fatal message and exit
func Fatalf(format string, args ...interface{}) {
	std.Fatalf(format, args...)
}

// WithFields setter context to log message
func WithFields(v Fields) Writer {
	return std.WithFields(v)
}

// WithError setter context to log message
func WithError(key string, err error) Writer {
	return std.WithError(key, err)
}

// WithField setter context to log message
func WithField(key string, value interface{}) Writer {
	return std.WithField(key, value)
}
