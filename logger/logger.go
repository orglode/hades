package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// LogLevel 自定义日志级别
type LogLevel int

const (
	Silent LogLevel = iota
	Error
	Warn
	Info
	Debug
)

// Logger 核心日志结构
type Logger struct {
	zapLogger *zap.Logger
	level     LogLevel
	mu        sync.Mutex
}

// NewLogger 创建日志实例
func NewLogger(logPath string, level LogLevel) (*Logger, error) {
	if logPath == "" {
		logPath = "./logs"
	}

	// 1. 强制检查日志目录
	if err := ensureLogDir(logPath); err != nil {
		return nil, err
	}

	// 2. 配置 rotatelogs（关键修改：确保文件打开成功）
	rl, err := rotatelogs.New(
		filepath.Join(logPath, "app-%Y-%m-%d.log"),
		rotatelogs.WithClock(rotatelogs.Local),
		rotatelogs.WithMaxAge(30*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithLinkName(filepath.Join(logPath, "app_current.log")), // 明确软链接路径
	)
	if err != nil {
		return nil, fmt.Errorf("创建rotatelogs失败: %w", err)
	}

	// 3. 立即测试文件写入
	if _, err := rl.Write([]byte("=== INIT LOGGER ===\n")); err != nil {
		return nil, fmt.Errorf("日志文件写入测试失败: %w", err)
	}

	// 4. 配置zap核心
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(rl),        // 主日志文件
			zapcore.AddSync(os.Stdout), // 同时输出到控制台
		),
		zapcore.Level(level),
	)

	// 5. 构建Logger（添加文件写入确认钩子）
	zapLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Hooks(func(entry zapcore.Entry) error {
			// 每次写入后同步文件（生产环境建议异步处理）
			_ = rl.Rotate()
			return nil
		}),
	)

	return &Logger{
		zapLogger: zapLogger,
		level:     level,
	}, nil
}

// 实现日志级别方法
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zapLogger.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zapLogger.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zapLogger.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zapLogger.Error(msg, fields...)
}

func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.zapLogger.Panic(msg, fields...)
}

// WithFields 添加结构化字段
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	return &Logger{
		zapLogger: l.zapLogger.With(fields...),
		level:     l.level,
	}
}

// Sync 刷新缓冲区的日志
func (l *Logger) Sync() error {
	return l.zapLogger.Sync()
}

// GormLogger GORM日志适配器
type GormLogger struct {
	Logger                  *Logger
	SlowThreshold           time.Duration
	SkipCallerLookup        bool
	IgnoreRecordNotFoundErr bool
}

// NewGormLogger 创建GORM日志适配器
func NewGormLogger(logger *Logger) *GormLogger {
	return &GormLogger{
		Logger:                  logger,
		SlowThreshold:           200 * time.Millisecond,
		SkipCallerLookup:        false,
		IgnoreRecordNotFoundErr: true,
	}
}

// LogMode 实现gorm logger.Interface接口
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	switch level {
	case gormlogger.Silent:
		newLogger.Logger.level = Silent
	case gormlogger.Error:
		newLogger.Logger.level = Error
	case gormlogger.Warn:
		newLogger.Logger.level = Warn
	case gormlogger.Info:
		newLogger.Logger.level = Info
	}
	return &newLogger
}

// Info 实现gorm logger.Interface接口
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.Logger.level >= Info {
		l.Logger.Info(fmt.Sprintf(msg, data...))
	}
}

// Warn 实现gorm logger.Interface接口
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.Logger.level >= Warn {
		l.Logger.Warn(fmt.Sprintf(msg, data...))
	}
}

// Error 实现gorm logger.Interface接口
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.Logger.level >= Error {
		l.Logger.Error(fmt.Sprintf(msg, data...))
	}
}

// Trace 实现gorm logger.Interface接口
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.Logger.level <= Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
	}

	switch {
	case err != nil && !(l.IgnoreRecordNotFoundErr && err == gorm.ErrRecordNotFound):
		l.Logger.Error("gorm error", append(fields, zap.Error(err))...)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		l.Logger.Warn(fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold), fields...)
	case l.Logger.level == Debug:
		l.Logger.Debug("gorm trace", fields...)
	}
}

// Recover 捕获panic
func (l *Logger) Recover() {
	if err := recover(); err != nil {
		stack := make([]byte, 4096)
		length := runtime.Stack(stack, false)
		l.Error("PANIC RECOVERED",
			zap.Any("error", err),
			zap.String("stack", string(stack[:length])),
		)
	}
}

func ensureLogDir(logPath string) error {
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 验证目录是否可写
	testFile := filepath.Join(logPath, ".testwrite")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("日志目录不可写: %w", err)
	}
	_ = os.Remove(testFile) // 清理测试文件

	return nil
}
