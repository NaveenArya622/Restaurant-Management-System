package log

import (
	"context"
	"github.com/sirupsen/logrus"
)

var Logger *loggerStruct

type loggerStruct struct {
	*logrus.Logger
}

type entry struct {
	*logrus.Entry
}

type Fields logrus.Fields

type logInterface interface {
	DebugWithContext(context context.Context, args ...interface{})
	Debug(args ...interface{})
	InfoWithContext(context context.Context, args ...interface{})
	Info(args ...interface{})
	WarnWithContext(context context.Context, args ...interface{})
	Warn(args ...interface{})
	ErrorWithContext(context context.Context, args ...interface{})
	Error(args ...interface{})
	FatalWithContext(context context.Context, args ...interface{})
	Fatal(args ...interface{})
}

func init() {
	loggerInstance := logrus.New()
	loggerInstance.Formatter = &logrus.JSONFormatter{}
	loggerInstance.Level = logrus.DebugLevel
	Logger = &loggerStruct{loggerInstance}
}

func (logger *loggerStruct) WithContext(context context.Context) *entry {
	return &entry{
		logger.Logger.WithField("requestId", context.Value("requestId")),
	}
}

func (logger *loggerStruct) DebugWithContext(context context.Context, args ...interface{}) {
	logger.WithContext(context).Debug(args...)
}

func (logger *loggerStruct) Debug(args ...interface{}) {
	logger.Log(logrus.DebugLevel, args...)
}

func (logger *loggerStruct) InfoWithContext(context context.Context, args ...interface{}) {
	logger.WithContext(context).Info(args...)
}

func (logger *loggerStruct) Info(args ...interface{}) {
	logger.Log(logrus.InfoLevel, args...)
}

func (logger *loggerStruct) WarnWithContext(context context.Context, args ...interface{}) {
	logger.WithContext(context).Warn(args...)
}

func (logger *loggerStruct) Warn(args ...interface{}) {
	logger.Log(logrus.WarnLevel, args...)
}

func (logger *loggerStruct) ErrorWithContext(context context.Context, args ...interface{}) {
	logger.WithContext(context).Error(args...)
}

func (logger *loggerStruct) Error(args ...interface{}) {
	logger.Log(logrus.ErrorLevel, args...)
}

func (logger *loggerStruct) FatalWithContext(context context.Context, args ...interface{}) {
	logger.WithContext(context).Fatal(args...)
}

func (logger *loggerStruct) Fatal(args ...interface{}) {
	logger.Log(logrus.FatalLevel, args...)
}

func (logger *loggerStruct) WithField(key string, value interface{}) *entry {
	return &entry{
		logger.Logger.WithFields(logrus.Fields{key: value}),
	}
}

func (logger *loggerStruct) WithFields(fields Fields) *entry {
	return &entry{
		logger.Logger.WithFields(logrus.Fields(fields)),
	}
}

func (e *entry) DebugWithContext(context context.Context, args ...interface{}) {
	e.WithContext(context).Debug(args...)
}

func (e *entry) Debug(args ...interface{}) {
	e.Log(logrus.DebugLevel, args...)
}

func (e *entry) InfoWithContext(context context.Context, args ...interface{}) {
	e.WithContext(context).Info(args...)
}

func (e *entry) Info(args ...interface{}) {
	e.Log(logrus.InfoLevel, args...)
}

func (e *entry) WarnWithContext(context context.Context, args ...interface{}) {
	e.WithContext(context).Warn(args...)
}

func (e *entry) Warn(args ...interface{}) {
	e.Log(logrus.WarnLevel, args...)
}

func (e *entry) ErrorWithContext(context context.Context, args ...interface{}) {
	e.WithContext(context).Error(args...)
}

func (e *entry) Error(args ...interface{}) {
	e.Log(logrus.ErrorLevel, args...)
}

func (e *entry) FatalWithContext(context context.Context, args ...interface{}) {
	e.WithContext(context).Fatal(args...)
}

func (e *entry) Fatal(args ...interface{}) {
	e.Log(logrus.FatalLevel, args...)
}
