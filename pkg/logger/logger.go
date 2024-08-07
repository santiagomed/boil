package logger

type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)
	WithField(key string, value interface{}) Logger
}

type NullLogger struct{}

func (NullLogger) Debug(msg string) {}
func (NullLogger) Info(msg string)  {}
func (NullLogger) Warn(msg string)  {}
func (NullLogger) Error(msg string) {}
func (NullLogger) Fatal(msg string) {}
func (NullLogger) WithField(key string, value interface{}) Logger {
	return NullLogger{}
}

func NewNullLogger() Logger {
	return NullLogger{}
}
