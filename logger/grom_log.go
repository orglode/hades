package logger

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/orglode/hades/trace"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// GormLogger 返回 GORM 的日志器
func GormLogger() logger.Interface {
	if globalLogger == nil || globalLogger.sqlLogger == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return logger.Default
	}
	return &gormLogger{logger: globalLogger}
}

// gormLogger 实现 GORM 的 logger.Interface
type gormLogger struct {
	logger *Logger
}

// LogMode 设置 GORM 日志模式
func (g *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

// getCallerInfo 获取调用栈信息（文件名和行号）
func getCallerInfo() (string, int) {
	_, file, line, ok := runtime.Caller(4) // 跳过 4 层调用栈以获取实际调用 GORM 的位置
	if !ok {
		return "unknown", 0
	}
	return file, line
}

// buildFields 构建日志字段，确保 SQL 语句放在前面
func buildFields(ctx context.Context, sql string, rows int64, elapsed time.Duration, err error) []zap.Field {
	fields := []zap.Field{
		zap.String("sql", sql), // SQL 语句放在最前面
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	if file, line := getCallerInfo(); file != "unknown" {
		fields = append(fields, zap.String("file", file), zap.Int("line", line))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	return fields
}

// Info 记录 GORM Info 日志
func (g *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	fields := []zap.Field{}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	if file, line := getCallerInfo(); file != "unknown" {
		fields = append(fields, zap.String("file", file), zap.Int("line", line))
	}
	g.logger.sqlLogger.Info(fmt.Sprintf(msg, data...), fields...)
}

// Warn 记录 GORM Warn 日志
func (g *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	fields := []zap.Field{}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	if file, line := getCallerInfo(); file != "unknown" {
		fields = append(fields, zap.String("file", file), zap.Int("line", line))
	}
	g.logger.sqlLogger.Warn(fmt.Sprintf(msg, data...), fields...)
}

// Error 记录 GORM Error 日志
func (g *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	fields := []zap.Field{}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	if file, line := getCallerInfo(); file != "unknown" {
		fields = append(fields, zap.String("file", file), zap.Int("line", line))
	}
	g.logger.sqlLogger.Error(fmt.Sprintf(msg, data...), fields...)
}

// Trace 记录 GORM SQL 执行日志
func (g *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := buildFields(ctx, sql, rows, elapsed, err)

	if err != nil {
		g.logger.sqlLogger.Error("sql execution error", fields...)
	} else {
		g.logger.sqlLogger.Debug("sql executed", fields...)
	}
}
