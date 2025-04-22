package log

import (
	log "github.com/sirupsen/logrus"
)

const (
	Trace Level = Level(log.TraceLevel)
	Debug Level = Level(log.DebugLevel)
	Info  Level = Level(log.InfoLevel)
	Warn  Level = Level(log.WarnLevel)
	Error Level = Level(log.ErrorLevel)
)

type Level int32

type logger struct {
	log *log.Logger
}

func New(level Level) *logger {
	l := log.New()
	l.SetLevel(log.Level(level))

	return &logger{log: l}
}

func (l *logger) Trace(msg string) {
	l.log.Trace(msg)
}

func (l *logger) Debug(msg string) {
	l.log.Debug(msg)
}

func (l *logger) Info(msg string) {
	l.log.Info(msg)
}

func (l *logger) Warn(msg string) {
	l.log.Warn(msg)
}

func (l *logger) Error(err error) {
	l.log.Error(err.Error())
}

func (l *logger) Fatal(err error) {
	l.log.Fatal(err)
}
