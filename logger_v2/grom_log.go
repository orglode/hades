package logger_v2

import (
	"context"
	"errors"
	"time"

	"github.com/orglode/hades/trace"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormLoggerAdapter 适配器，将zap.Logger适配到GORM的logger.Interface
type GormLoggerAdapter struct {
	*zap.Logger
	LogLevel logger.LogLevel
}

// NewGormLogger 创建一个GORM日志适配器
func NewGormLogger(zapLogger *zap.Logger) logger.Interface {
	return &GormLoggerAdapter{
		Logger:   zapLogger,
		LogLevel: logger.Info, // 默认级别
	}
}

// LogMode 设置日志级别
func (g *GormLoggerAdapter) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *g
	newLogger.LogLevel = level
	return &newLogger
}

// Info 打印信息
func (g *GormLoggerAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= logger.Info {
		g.Logger.Info(msg, zap.Any("data", data))
	}
}

// Warn 打印警告
func (g *GormLoggerAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= logger.Warn {
		g.Logger.Warn(msg, zap.Any("data", data))
	}
}

// Error 打印错误
func (g *GormLoggerAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if g.LogLevel >= logger.Error {
		g.Logger.Error(msg, zap.Any("data", data))
	}
}

// Trace 实现GORM的Trace方法
func (g *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if g.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.Time("begin", begin),
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	fields = append(fields, zap.String("caller", getSimplifiedCaller(4)))

	if err != nil && g.LogLevel >= logger.Error {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			fields = append(fields, zap.Error(err))
			g.Logger.Error("SQL执行错误", fields...)
		}
	} else if elapsed > 200*time.Millisecond && g.LogLevel >= logger.Warn {
		fields = append(fields, zap.Duration("slow_threshold", 200*time.Millisecond))
		g.Logger.Warn("SQL慢查询", fields...)
	} else if g.LogLevel >= logger.Info {
		g.Logger.Info("SQL查询", fields...)
	}
}
