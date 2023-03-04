package logging

import (
	"context"
	"go.uber.org/zap"
)

var loggings map[string]*Logger

const (
	initLevelInfo   = "info"
	initLevelError  = "error"
	initLevelDebug  = "debug"
	initLevelAccess = "access"
	initLevelCommon = "common"
)

func init() {
	loggings = map[string]*Logger{
		"error":  newInitLogger(initLevelError),
		"info":   newInitLogger(initLevelInfo),
		"access": newInitLogger(initLevelAccess),
		"debug":  newInitLogger(initLevelDebug),
	}
}

type Logger struct {
	defaultLogging *zap.SugaredLogger
}

func newInitLogger(initLevelConf string) *Logger {
	return &Logger{
		defaultLogging: zap.New(initCoreEncoder(initLevelConf)).WithOptions(zap.AddCaller(), zap.AddCallerSkip(1)).Sugar(),
	}
}

func NewLogger() *Logger {
	return &Logger{
		defaultLogging: zap.New(initCoreEncoder(initLevelCommon)).WithOptions(zap.AddCaller(), zap.AddCallerSkip(1)).Sugar(),
	}
}

func For(ctx context.Context) *Logger {

	return NewLogger()
}

func (l *Logger) Error(params ...interface{}) {
	l.defaultLogging.Error(params...)
}

func (l *Logger) Info(params ...interface{}) {
	l.defaultLogging.Info(params...)
}

func (l *Logger) Debug(params ...interface{}) {
	l.defaultLogging.Debug(params...)
}
func (l *Logger) Panic(params ...interface{}) {
	l.defaultLogging.Panic(params...)
}
func (l *Logger) DPanic(params ...interface{}) {
	l.defaultLogging.DPanic(params...)
}

func Errorf(key string, params ...interface{}) {
	loggings[initLevelError].Errorf(key, params...)
}

func Infof(key string, params ...interface{}) {
	loggings[initLevelInfo].Infof(key, params...)
}

func Debugf(key string, params ...interface{}) {
	loggings[initLevelDebug].Debugf(key, params...)
}
func Accessf(key string, params ...interface{}) {
	loggings[initLevelAccess].Infof(key, params...)
}

func (l *Logger) Errorf(key string, params ...interface{}) {
	l.defaultLogging.Errorf(key, params...)
}

func (l *Logger) Infof(key string, params ...interface{}) {
	l.defaultLogging.Infof(key, params...)
}

func (l *Logger) Debugf(key string, params ...interface{}) {
	l.defaultLogging.Debugf(key, params...)
}
func (l *Logger) Panicf(key string, params ...interface{}) {
	l.defaultLogging.Panicf(key, params...)
}
