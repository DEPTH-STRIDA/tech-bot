package logger

import (
	"easycodeapp/internal/config"
	"easycodeapp/pkg/logger"
	"easycodeapp/pkg/logger/interfaces"
)

var Log interfaces.Logger

func init() {
	cfg := logger.Config{
		LogDir:         config.File.LoggerConfig.LogDir,
		LogMaxFileSize: config.File.LoggerConfig.MaxFileSize,
		LogTimeFormat:  config.File.LoggerConfig.TimeFormat,
		LogFilePattern: config.File.LoggerConfig.FilePattern,
	}

	var err error
	Log, err = logger.New(cfg)
	if err != nil {
		panic(err)
	}

}

// Методы без форматирования
func Info(args ...interface{})  { Log.Info(args...) }
func Error(args ...interface{}) { Log.Error(args...) }
func Debug(args ...interface{}) { Log.Debug(args...) }
func Warn(args ...interface{})  { Log.Warn(args...) }
func Fatal(args ...interface{}) { Log.Fatal(args...) }

// Методы с форматированием
func Infof(format string, args ...interface{})  { Log.Infof(format, args...) }
func Errorf(format string, args ...interface{}) { Log.Errorf(format, args...) }
func Debugf(format string, args ...interface{}) { Log.Debugf(format, args...) }
func Warnf(format string, args ...interface{})  { Log.Warnf(format, args...) }
func Fatalf(format string, args ...interface{}) { Log.Fatalf(format, args...) }

// Методы для ошибок со стектрейсом
func ErrorWithStack(err error, msg string) { Log.ErrorWithStack(err, msg) }
func ErrorWithStackf(err error, format string, args ...interface{}) {
	Log.ErrorWithStackf(err, format, args...)
}
