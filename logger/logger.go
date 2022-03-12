package logger

import "log"

type Logger interface {
	Debug(message string)
	Info(message string)
	Warn(message string)
	Error(message string)
}

type LogLevel uint

const (
	LogLevelDebug = LogLevel(100)
	LogLevelInfo  = LogLevel(60)
	LogLevelWarn  = LogLevel(40)
	LogLevelError = LogLevel(20)
	LogLevelNo    = LogLevel(0)
)

type DefaultLogger struct{}

func (l *DefaultLogger) Debug(message string) {
	log.Printf("DEBUG %s", message)
}
func (l *DefaultLogger) Info(message string) {
	log.Printf("INFO %s", message)
}
func (l *DefaultLogger) Warn(message string) {
	log.Printf("WARN %s", message)
}
func (l *DefaultLogger) Error(message string) {
	log.Printf("ERR %s", message)
}

func GetLevel(level string) LogLevel {
	m := map[string]LogLevel{
		"debug":   LogLevelDebug,
		"info":    LogLevelInfo,
		"warning": LogLevelWarn,
		"error":   LogLevelError,
		"no":      LogLevelNo,
	}
	if l, ok := m[level]; ok {
		return l
	}
	return LogLevelInfo
}
