package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"easycodeapp/pkg/logger/interfaces"

	"github.com/rs/zerolog"
)

// Config конфигурация для создания логгера
// Добавлены параметры для ротации логов
// LogDir - директория для логов
// LogMaxFileSize - максимальный размер файла лога в байтах
// LogTimeFormat - формат времени для имени файла
// LogFilePattern - шаблон имени файла

type Config struct {
	LogDir         string
	LogMaxFileSize int64
	LogTimeFormat  string
	LogFilePattern string
}

// ZerologLogger реализация логгера на основе zerolog
type ZerologLogger struct {
	log zerolog.Logger
}

// createLogFile создает файл для логирования с учетом ротации
func createLogFile(cfg Config) (io.Writer, error) {
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию для логов: %w", err)
	}

	logFilePath := filepath.Join(cfg.LogDir, fmt.Sprintf(cfg.LogFilePattern, time.Now().Format(cfg.LogTimeFormat)))

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл логов: %w", err)
	}

	// Проверка размера файла и ротация
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить информацию о файле: %w", err)
	}

	if fileInfo.Size() >= cfg.LogMaxFileSize {
		file.Close()
		newLogFilePath := filepath.Join(cfg.LogDir, fmt.Sprintf(cfg.LogFilePattern, time.Now().Format(cfg.LogTimeFormat)))
		file, err = os.OpenFile(newLogFilePath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("не удалось создать новый файл логов: %w", err)
		}
	}

	return file, nil
}

// New создает новый логгер на основе конфигурации
func New(cfg Config) (interfaces.Logger, error) {
	var fileWriter io.Writer
	var err error

	if cfg.LogDir != "" {
		fileWriter, err = createLogFile(cfg)
		if err != nil {
			return nil, err
		}
	} else {
		fileWriter = os.Stdout
	}

	// Создаем MultiWriter для записи в файл и консоль
	writer := io.MultiWriter(fileWriter, os.Stdout)

	l := &ZerologLogger{
		log: zerolog.New(writer).With().Timestamp().Logger(),
	}

	return l, nil
}

// Print реализация интерфейса BasicLogger
func (l *ZerologLogger) Print(v ...interface{}) {
	l.Info(v...)
}

func (l *ZerologLogger) Printf(format string, v ...interface{}) {
	l.Infof(format, v...)
}

func (l *ZerologLogger) Println(v ...interface{}) {
	l.Info(v...)
}

// Info реализация интерфейса LevelLogger
func (l *ZerologLogger) Info(args ...interface{}) {
	l.log.Info().Msg(fmt.Sprint(args...))
}

func (l *ZerologLogger) Error(args ...interface{}) {
	l.log.Error().Msg(fmt.Sprint(args...))
}

func (l *ZerologLogger) Debug(args ...interface{}) {
	l.log.Debug().Msg(fmt.Sprint(args...))
}

func (l *ZerologLogger) Warn(args ...interface{}) {
	l.log.Warn().Msg(fmt.Sprint(args...))
}

func (l *ZerologLogger) Fatal(args ...interface{}) {
	l.log.Fatal().Msg(fmt.Sprint(args...))
}

// Infof реализация интерфейса FormattedLevelLogger
func (l *ZerologLogger) Infof(format string, args ...interface{}) {
	l.log.Info().Msgf(format, args...)
}

func (l *ZerologLogger) Errorf(format string, args ...interface{}) {
	l.log.Error().Msgf(format, args...)
}

func (l *ZerologLogger) Debugf(format string, args ...interface{}) {
	l.log.Debug().Msgf(format, args...)
}

func (l *ZerologLogger) Warnf(format string, args ...interface{}) {
	l.log.Warn().Msgf(format, args...)
}

func (l *ZerologLogger) Fatalf(format string, args ...interface{}) {
	l.log.Fatal().Msgf(format, args...)
}

// ErrorWithStack реализация интерфейса StackTraceLogger
func (l *ZerologLogger) ErrorWithStack(err error, msg string) {
	l.log.Error().Stack().Err(err).Msg(msg)
}

func (l *ZerologLogger) ErrorWithStackf(err error, format string, args ...interface{}) {
	l.log.Error().Stack().Err(err).Msgf(format, args...)
}

// WithFields реализация интерфейса ContextLogger
func (l *ZerologLogger) WithFields(fields map[string]interface{}) interfaces.Logger {
	ctx := l.log.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &ZerologLogger{log: ctx.Logger()}
}
