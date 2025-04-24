package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 是日志实例
type Logger struct {
	config       Config
	syncers      map[string]zapcore.WriteSyncer // 按日志类型存储的写入器
	levelLoggers map[LogLevel]*otelzap.Logger   // 按日志级别存储的专用Logger
	accessLogger *otelzap.Logger                // Gin访问日志Logger
	sqlLogger    *otelzap.Logger                // GORM SQL日志Logger
}

// Config 日志配置
type Config struct {
	LogDir       string        // 日志目录
	MaxAge       time.Duration // 日志最大保留时间
	RotationTime time.Duration // 日志分割时间
	Level        string        // 日志级别
	JSONFormat   bool          // 是否使用JSON格式
}

// LogLevel 定义日志级别
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// CustomError 定义自定义错误结构
type CustomError struct {
	Code    string                 // 错误码
	Message string                 // 错误消息
	Fields  map[string]interface{} // 附加字段
}

// 全局日志器实例
var globalLogger *Logger

// NewCustomError 创建自定义错误
func NewCustomError(code, message string, fields map[string]interface{}) *CustomError {
	return &CustomError{
		Code:    code,
		Message: message,
		Fields:  fields,
	}
}

// InitLogger 初始化全局日志器
func InitLogger(config Config) error {
	// 设置默认值
	if config.LogDir == "" {
		config.LogDir = "./logs"
	}
	if config.MaxAge == 0 {
		config.MaxAge = 30 * 24 * time.Hour
	}
	if config.RotationTime == 0 {
		config.RotationTime = 24 * time.Hour
	}
	if config.Level == "" {
		config.Level = "info"
	}

	// 确保日志目录存在
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// 设置全局日志级别
	var zapLevel zapcore.Level
	switch config.Level {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	case "fatal":
		zapLevel = zap.FatalLevel
	default:
		zapLevel = zap.InfoLevel
	}

	// 配置编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if config.JSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 初始化不同类型的日志写入器和Logger
	syncers := make(map[string]zapcore.WriteSyncer)
	levelLoggers := make(map[LogLevel]*otelzap.Logger)

	// 日志级别对应的文件名和Zap级别映射
	levelFiles := map[LogLevel]struct {
		fileName string
		zapLevel zapcore.Level
	}{
		DebugLevel: {"debug_%Y%m%d.log", zapcore.DebugLevel},
		InfoLevel:  {"info_%Y%m%d.log", zapcore.InfoLevel},
		WarnLevel:  {"warn_%Y%m%d.log", zapcore.WarnLevel},
		ErrorLevel: {"error_%Y%m%d.log", zapcore.ErrorLevel},
		FatalLevel: {"fatal_%Y%m%d.log", zapcore.FatalLevel},
	}

	// 为每个日志级别创建rotatelogs写入器和专用Logger
	for level, info := range levelFiles {
		rotator, err := rotatelogs.New(
			filepath.Join(config.LogDir, info.fileName),
			rotatelogs.WithMaxAge(config.MaxAge),
			rotatelogs.WithRotationTime(config.RotationTime),
			rotatelogs.WithLinkName(filepath.Join(config.LogDir, info.fileName[:len(info.fileName)-len("_%Y%m%d.log")]+".log")),
		)
		if err != nil {
			return fmt.Errorf("failed to initialize rotatelogs for %s: %w", info.fileName, err)
		}
		syncer := zapcore.AddSync(rotator)
		syncers[fmt.Sprintf("level_%d", level)] = syncer

		// 创建仅允许特定级别的核心
		levelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl == info.zapLevel && lvl >= zapLevel
		})
		core := zapcore.NewCore(encoder, syncer, levelEnabler)
		zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
		levelLoggers[level] = otelzap.New(zapLogger)
	}

	// 创建Gin访问日志的rotatelogs写入器和Logger
	accessRotator, err := rotatelogs.New(
		filepath.Join(config.LogDir, "access_%Y%m%d.log"),
		rotatelogs.WithMaxAge(config.MaxAge),
		rotatelogs.WithRotationTime(config.RotationTime),
		rotatelogs.WithLinkName(filepath.Join(config.LogDir, "access.log")),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize rotatelogs for access: %w", err)
	}
	accessSyncer := zapcore.AddSync(accessRotator)
	syncers["access"] = accessSyncer
	accessCore := zapcore.NewCore(encoder, accessSyncer, zapLevel)
	accessZapLogger := zap.New(accessCore, zap.AddCaller())
	accessLogger := otelzap.New(accessZapLogger)

	// 创建GORM SQL日志的rotatelogs写入器和Logger
	sqlRotator, err := rotatelogs.New(
		filepath.Join(config.LogDir, "sql_%Y%m%d.log"),
		rotatelogs.WithMaxAge(config.MaxAge),
		rotatelogs.WithRotationTime(config.RotationTime),
		rotatelogs.WithLinkName(filepath.Join(config.LogDir, "sql.log")),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize rotatelogs for sql: %w", err)
	}
	sqlSyncer := zapcore.AddSync(sqlRotator)
	syncers["sql"] = sqlSyncer
	sqlCore := zapcore.NewCore(encoder, sqlSyncer, zapLevel)
	sqlZapLogger := zap.New(sqlCore, zap.AddCaller())
	sqlLogger := otelzap.New(sqlZapLogger)

	globalLogger = &Logger{
		config:       config,
		syncers:      syncers,
		levelLoggers: levelLoggers,
		accessLogger: accessLogger,
		sqlLogger:    sqlLogger,
	}
	return nil
}

// Debug 记录Debug级别日志
func Debug(msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[DebugLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	globalLogger.levelLoggers[DebugLevel].Debug(msg, fields...)
}

// Info 记录Info级别日志
func Info(msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[InfoLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	globalLogger.levelLoggers[InfoLevel].Info(msg, fields...)
}

// Warn 记录Warn级别日志
func Warn(msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[WarnLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	globalLogger.levelLoggers[WarnLevel].Warn(msg, fields...)
}

// Error 记录Error级别日志
func Error(msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[ErrorLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	globalLogger.levelLoggers[ErrorLevel].Error(msg, fields...)
}

// Fatal 记录Fatal级别日志
func Fatal(msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[FatalLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		os.Exit(1)
	}
	globalLogger.levelLoggers[FatalLevel].Fatal(msg, fields...)
}

// LogCustomError 记录自定义错误
func LogCustomError(customErr *CustomError) {
	if globalLogger == nil || globalLogger.levelLoggers[ErrorLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	fields := []zap.Field{
		zap.String("error_code", customErr.Code),
	}
	for k, v := range customErr.Fields {
		fields = append(fields, zap.Any(k, v))
	}
	globalLogger.levelLoggers[ErrorLevel].Error(customErr.Message, fields...)
}

// Sync 同步日志缓冲区
func Sync() error {
	if globalLogger == nil {
		return fmt.Errorf("logger not initialized")
	}
	var lastErr error
	for level, l := range globalLogger.levelLoggers {
		if err := l.Sync(); err != nil {
			lastErr = fmt.Errorf("failed to sync level %d logger: %w", level, err)
		}
	}
	if err := globalLogger.accessLogger.Sync(); err != nil {
		lastErr = fmt.Errorf("failed to sync access logger: %w", err)
	}
	if err := globalLogger.sqlLogger.Sync(); err != nil {
		lastErr = fmt.Errorf("failed to sync sql logger: %w", err)
	}
	return lastErr
}

// Close 关闭日志器
func Close() error {
	return Sync()
}
