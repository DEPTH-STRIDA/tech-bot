package logger

import (
	"easycodeapp/pkg/logger/interfaces"
	"fmt"

	"github.com/rs/zerolog"
)

// ZerologEvent обертка над событием zerolog
type ZerologEvent struct {
	event *zerolog.Event
}

func (e *ZerologEvent) Info(args ...interface{}) {
	e.event.Msg(fmt.Sprint(args...))
}

func (e *ZerologEvent) Infof(format string, args ...interface{}) {
	e.event.Msgf(format, args...)
}

func (e *ZerologEvent) Error(args ...interface{}) {
	e.event.Msg(fmt.Sprint(args...))
}

func (e *ZerologEvent) Errorf(format string, args ...interface{}) {
	e.event.Msgf(format, args...)
}

func (e *ZerologEvent) Debug(args ...interface{}) {
	e.event.Msg(fmt.Sprint(args...))
}

func (e *ZerologEvent) Debugf(format string, args ...interface{}) {
	e.event.Msgf(format, args...)
}

func (e *ZerologEvent) Warn(args ...interface{}) {
	e.event.Msg(fmt.Sprint(args...))
}

func (e *ZerologEvent) Warnf(format string, args ...interface{}) {
	e.event.Msgf(format, args...)
}

func (e *ZerologEvent) Fatal(args ...interface{}) {
	e.event.Msg(fmt.Sprint(args...))
}

func (e *ZerologEvent) Fatalf(format string, args ...interface{}) {
	e.event.Msgf(format, args...)
}

func (e *ZerologEvent) Print(v ...interface{}) {
	e.Info(v...)
}

func (e *ZerologEvent) Printf(format string, v ...interface{}) {
	e.Infof(format, v...)
}

func (e *ZerologEvent) Println(v ...interface{}) {
	e.Info(v...)
}

func (e *ZerologEvent) ErrorWithStack(err error, msg string) {
	e.event.Stack().Err(err).Msg(msg)
}

func (e *ZerologEvent) ErrorWithStackf(err error, format string, args ...interface{}) {
	e.event.Stack().Err(err).Msgf(format, args...)
}

func (e *ZerologEvent) WithFields(fields map[string]interface{}) interfaces.Logger {
	for k, v := range fields {
		e.event.Interface(k, v)
	}
	return e
}
