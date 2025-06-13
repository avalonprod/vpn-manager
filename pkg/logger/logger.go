package logger

import "github.com/sirupsen/logrus"

type ILogger interface {
	Debug(msg ...interface{})
	Debugf(format string, args ...interface{})
	Info(msg ...interface{})
	Infof(format string, args ...interface{})
	Warn(msg ...interface{})
	Warnf(format string, args ...interface{})
	Error(msg ...interface{})
	Errorf(format string, args ...interface{})
}

type logger struct {
}

func NewLogger() ILogger {
	return &logger{}
}

func (l *logger) Debug(msg ...interface{}) {
	logrus.Debug(msg...)
}

func (l *logger) Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

func (l *logger) Info(msg ...interface{}) {
	logrus.Info(msg...)
}

func (l *logger) Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}
func (l *logger) Warn(msg ...interface{}) {
	logrus.Warn(msg...)
}

func (l *logger) Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

func (l *logger) Error(msg ...interface{}) {
	logrus.Error(msg...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}
