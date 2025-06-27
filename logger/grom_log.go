package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/orglode/hades/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
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
	logger      *Logger
	sqlLogger   *zap.Logger // 用于 Info, Warn, Trace 日志，写入 YYYYMMDD.log 和终端
	errorLogger *zap.Logger // 用于 Error 日志，写入 error_YYYYMMDD.log
}

// initSQLLogger 初始化 Info, Warn, Trace 级别的日志器
func initSQLLogger() *zap.Logger {
	logFile := time.Now().Format("20060102") + ".log"
	writeSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: 5,
		MaxAge:     30, // 天
		Compress:   true,
	})
	// 同时写入文件和终端
	multiSyncer := zapcore.NewMultiWriteSyncer(writeSyncer, zapcore.AddSync(os.Stdout))
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		multiSyncer,
		zap.DebugLevel, // 支持 Debug, Info, Warn, Error
	)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
}

// initErrorLogger 初始化 Error 级别的专用日志器
func initErrorLogger() *zap.Logger {
	logFile := fmt.Sprintf("error_%s.log", time.Now().Format("20060102"))
	writeSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: 5,
		MaxAge:     30, // 天
		Compress:   true,
	})
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		writeSyncer,
		zap.ErrorLevel,
	)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
}

// LogMode 设置 GORM 日志模式
func (g *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

// getCallerInfo 获取业务代码的调用栈信息（文件名和行号）
func getCallerInfo() (string, int) {
	for i := 4; i < 15; i++ { // 从第 4 层开始，最多检查 15 层
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			return "unknown", 0
		}
		// 过滤掉 GORM 内部路径（包含 vendor/gorm.io 或 gorm.io）
		if !strings.Contains(file, "gorm.io/gorm") {
			return file, line
		}
	}
	return "unknown", 0
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
		fields = append(fields, zap.String("file", filepath.Base(file)), zap.Int("line", line))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	return fields
}

// Info 记录 GORM Info 日志
func (g *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if g.sqlLogger == nil {
		g.sqlLogger = initSQLLogger()
	}
	fields := []zap.Field{}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	if file, line := getCallerInfo(); file != "unknown" {
		fields = append(fields, zap.String("file", filepath.Base(file)), zap.Int("line", line))
	}
	g.sqlLogger.Info(fmt.Sprintf(msg, data...), fields...)
}

// Warn 记录 GORM Warn 日志
func (g *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if g.sqlLogger == nil {
		g.sqlLogger = initSQLLogger()
	}
	fields := []zap.Field{}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	if file, line := getCallerInfo(); file != "unknown" {
		fields = append(fields, zap.String("file", filepath.Base(file)), zap.Int("line", line))
	}
	g.sqlLogger.Warn(fmt.Sprintf(msg, data...), fields...)
}

// Error 记录 GORM Error 日志
func (g *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if g.sqlLogger == nil {
		g.sqlLogger = initSQLLogger()
	}
	if g.errorLogger == nil {
		g.errorLogger = initErrorLogger()
	}
	fields := []zap.Field{}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("traceID", traceID))
	}
	if file, line := getCallerInfo(); file != "unknown" {
		fields = append(fields, zap.String("file", filepath.Base(file)), zap.Int("line", line))
	}
	// 写入 sqlLogger（20250627.log 和终端）
	g.sqlLogger.Error(fmt.Sprintf(msg, data...), fields...)
	// 写入 errorLogger（error_20250627.log）
	g.errorLogger.Error(fmt.Sprintf(msg, data...), fields...)
}

// Trace 记录 GORM SQL 执行日志
func (g *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if g.sqlLogger == nil {
		g.sqlLogger = initSQLLogger()
	}
	if err != nil && g.errorLogger == nil {
		g.errorLogger = initErrorLogger()
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := buildFields(ctx, sql, rows, elapsed, err)

	if err != nil {
		// 写入 sqlLogger（20250627.log 和终端）
		g.sqlLogger.Error("sql execution error", fields...)
		// 写入 errorLogger（error_20250627.log）
		g.errorLogger.Error("sql execution error", fields...)
	} else {
		g.sqlLogger.Debug("sql executed", fields...)
	}
}

// Sync 确保日志写入完成（用于优雅停止）
func (g *gormLogger) Sync() error {
	var errs []error
	if g.sqlLogger != nil {
		if err := g.sqlLogger.Sync(); err != nil {
			errs = append(errs, fmt.Errorf("failed to sync sqlLogger: %w", err))
		}
	}
	if g.errorLogger != nil {
		if err := g.errorLogger.Sync(); err != nil {
			errs = append(errs, fmt.Errorf("failed to sync errorLogger: %w", err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("sync errors: %v", errs)
	}
	return nil
}
