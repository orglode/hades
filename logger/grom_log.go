package logger

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
	"os"
	"time"
)

// GormLogger 返回GORM的日志器
func GormLogger() logger.Interface {
	if globalLogger == nil || globalLogger.sqlLogger == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return logger.Default
	}
	return &gormLogger{logger: globalLogger}
}

// gormLogger 实现GORM的logger.Interface
type gormLogger struct {
	logger *Logger
}

// LogMode 设置GORM日志模式
func (g *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

// Info 记录GORM Info日志
func (g *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	fields := []zap.Field{}
	if traceID := getTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	// GORM Info日志写入sql_*.log和终端
	g.logger.sqlLogger.Ctx(ctx).Info(fmt.Sprintf(msg, data...), fields...)
}

// Warn 记录GORM Warn日志
func (g *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	fields := []zap.Field{}
	if traceID := getTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	// GORM Warn日志写入sql_*.log和终端
	g.logger.sqlLogger.Ctx(ctx).Warn(fmt.Sprintf(msg, data...), fields...)
}

// Error 记录GORM Error日志
func (g *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	fields := []zap.Field{}
	if traceID := getTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	// GORM Error日志写入sql_*.log和终端
	g.logger.sqlLogger.Ctx(ctx).Error(fmt.Sprintf(msg, data...), fields...)
}

// Trace 记录GORM SQL执行日志
func (g *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}
	if traceID := getTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}

	if err != nil {
		// SQL错误日志写入sql_*.log和终端
		g.logger.sqlLogger.Ctx(ctx).Error("sql execution error", append(fields, zap.Error(err))...)
	} else {
		// SQL调试日志写入sql_*.log和终端
		g.logger.sqlLogger.Ctx(ctx).Debug("sql executed", fields...)
	}
}
