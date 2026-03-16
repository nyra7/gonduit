package log

type Logger interface {
	Write(msg string)
	Debug(msg string)
	Debugf(format string, args ...any)
	Error(msg string)
	Errorf(format string, args ...any)
	Warn(msg string)
	Warnf(format string, args ...any)
	Success(msg string)
	Successf(format string, args ...any)
	Danger(msg string)
	Dangerf(format string, args ...any)
	Info(msg string)
	Infof(format string, args ...any)
	Clear()
}
