package interfaces

// BasicLogger определяет базовый интерфейс для простого логирования.
type BasicLogger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// LevelLogger определяет интерфейс для логирования с уровнями.
type LevelLogger interface {
	Info(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})
	Warn(args ...interface{})
	Fatal(args ...interface{})
}

// FormattedLevelLogger определяет интерфейс для форматированного логирования с уровнями.
type FormattedLevelLogger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// StackTraceLogger определяет интерфейс для логирования со стектрейсом.
type StackTraceLogger interface {
	ErrorWithStack(err error, msg string)
	ErrorWithStackf(err error, format string, args ...interface{})
}

// ContextLogger определяет интерфейс для логирования с дополнительными полями контекста.
type ContextLogger interface {
	WithFields(fields map[string]interface{}) Logger
}

// Logger объединяет все интерфейсы логирования.
type Logger interface {
	BasicLogger
	LevelLogger
	FormattedLevelLogger
	StackTraceLogger
	ContextLogger
}

// SimpleLogger объединяет только базовые интерфейсы без стектрейса и контекста.
type SimpleLogger interface {
	BasicLogger
	LevelLogger
	FormattedLevelLogger
}
