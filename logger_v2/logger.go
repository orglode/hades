package logger_v2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/orglode/hades/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// Logger 是日志实例
type Logger struct {
	config       Config
	syncers      map[string]zapcore.WriteSyncer // 按日志类型存储的写入器
	levelLoggers map[LogLevel]*zap.Logger       // 按日志级别存储的专用Logger
	accessLogger *zap.Logger                    // Gin访问日志Logger
	sqlLogger    *zap.Logger                    // SQL日志Logger
}

// Config 日志配置
type Config struct {
	LogDir       string        // 日志目录
	MaxAge       time.Duration // 日志最大保留时间
	RotationTime time.Duration // 日志分割时间
	Level        string        // 日志级别
	JSONFormat   bool          // 是否使用JSON格式
	CallerSkip   int           // caller跳过的层数
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

// GinRequestInfo Gin请求信息结构
type GinRequestInfo struct {
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	Query        string            `json:"query,omitempty"`
	Status       int               `json:"status"`
	IP           string            `json:"ip"`
	UserAgent    string            `json:"user_agent"`
	Latency      string            `json:"latency"`
	RequestBody  interface{}       `json:"request_body,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	ResponseSize int               `json:"response_size"`
	Timestamp    string            `json:"timestamp"`
	TraceID      string            `json:"trace_id,omitempty"`
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

// 简化的caller编码器
func simpleCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	if !caller.Defined {
		enc.AppendString("???")
		return
	}
	_, file := filepath.Split(caller.File)
	enc.AppendString(fmt.Sprintf("%s:%d", file, caller.Line))
}

// 获取简化的堆栈信息
func getSimplifiedStack(skip int) string {
	pc := make([]uintptr, 10)
	n := runtime.Callers(skip, pc)
	if n == 0 {
		return ""
	}

	pc = pc[:n]
	frames := runtime.CallersFrames(pc)

	var simplified []string
	stackDepth := 0
	maxDepth := 3 // 最多显示3层栈

	for {
		frame, more := frames.Next()
		// 跳过标准库、vendor包和zap相关的调用
		if !strings.Contains(frame.File, "/vendor/") &&
			!strings.Contains(frame.Function, "go.uber.org/zap") &&
			!strings.Contains(frame.Function, "runtime.") &&
			!strings.Contains(frame.Function, "logger.") {

			_, file := filepath.Split(frame.File)
			funcName := frame.Function

			// 简化函数名
			if idx := strings.LastIndex(funcName, "/"); idx != -1 {
				funcName = funcName[idx+1:]
			}
			if idx := strings.Index(funcName, "."); idx != -1 {
				funcName = funcName[idx+1:]
			}

			simplified = append(simplified, fmt.Sprintf("%s:%d", file, frame.Line))
			stackDepth++
			if stackDepth >= maxDepth {
				break
			}
		}
		if !more {
			break
		}
	}

	if len(simplified) == 0 {
		return ""
	}
	return strings.Join(simplified, " > ")
}

// simplifiedStackEncoder 简化stacktrace的编码器
type simplifiedStackEncoder struct {
	zapcore.Encoder
	callerSkip int
}

func (s *simplifiedStackEncoder) Clone() zapcore.Encoder {
	return &simplifiedStackEncoder{
		Encoder:    s.Encoder.Clone(),
		callerSkip: s.callerSkip,
	}
}

func (s *simplifiedStackEncoder) EncodeEntry(entry zapcore.Entry, fields []zap.Field) (*buffer.Buffer, error) {
	// 在编码前简化stacktrace字段
	for i := range fields {
		if fields[i].Key == "stacktrace" {
			if stack, ok := fields[i].Interface.(string); ok {
				simplifiedStack := simplifyStackTrace(stack, s.callerSkip)
				fields[i].Interface = simplifiedStack
			}
		}
	}
	return s.Encoder.EncodeEntry(entry, fields)
}

// simplifyStackTrace 简化堆栈跟踪信息
func simplifyStackTrace(stack string, skip int) string {
	if stack == "" {
		return ""
	}

	lines := strings.Split(stack, "\n")
	if len(lines) <= 1 {
		return stack
	}

	var simplified []string
	stackDepth := 0
	maxDepth := 3 // 最多显示3层栈

	// 跳过第一个goroutine行
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// 检查是否是有效的栈帧行
		if strings.Contains(line, ".go:") {
			// 跳过标准库、vendor包和zap相关的调用
			if strings.Contains(line, "/vendor/") ||
				strings.Contains(line, "go.uber.org/zap") ||
				strings.Contains(line, "runtime.") ||
				strings.Contains(line, "github.com/orglode/") {
				continue
			}

			// 提取文件名和行号
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				// 格式如: /path/to/file.go:123
				filePath := parts[len(parts)-1]
				// 只取文件名
				_, file := filepath.Split(filePath)
				if file != "" {
					// 提取函数名
					funcName := ""
					if len(parts) > 1 {
						// 格式如: github.com/user/repo/package.function
						fullFunc := parts[0]
						if idx := strings.LastIndex(fullFunc, "/"); idx != -1 {
							funcName = fullFunc[idx+1:]
							if dotIdx := strings.Index(funcName, "."); dotIdx != -1 {
								funcName = funcName[dotIdx+1:]
							}
						}
					}

					if funcName != "" {
						simplified = append(simplified, fmt.Sprintf("%s(%s)", funcName, file))
					} else {
						simplified = append(simplified, file)
					}
					stackDepth++
					if stackDepth >= maxDepth {
						break
					}
				}
			}
		}
	}

	if len(simplified) == 0 {
		return "stacktrace: [simplified]"
	}
	return "stacktrace: " + strings.Join(simplified, " > ")
}

// 创建带简化stacktrace的编码器
func createEncoderWithSimpleStack(jsonFormat bool, callerSkip int) zapcore.Encoder {
	var baseEncoder zapcore.Encoder

	if jsonFormat {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.LevelKey = "level"
		encoderConfig.MessageKey = "msg"
		encoderConfig.CallerKey = "caller"
		encoderConfig.StacktraceKey = "stacktrace"
		encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.UTC().Format("2006-01-02 15:04:05"))
		}
		encoderConfig.EncodeCaller = simpleCallerEncoder
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder

		baseEncoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:       "T",
			LevelKey:      "L",
			NameKey:       "",
			CallerKey:     "C",
			MessageKey:    "M",
			StacktraceKey: "S",
			LineEnding:    zapcore.DefaultLineEnding,
			EncodeLevel:   zapcore.CapitalColorLevelEncoder,
			EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendString(t.Format("15:04:05"))
			},
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   simpleCallerEncoder,
		}
		baseEncoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	return &simplifiedStackEncoder{
		Encoder:    baseEncoder,
		callerSkip: callerSkip + 2, // 额外跳过几层
	}
}

// 获取日志级别
func getZapLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// 创建级别日志器
func createLevelLoggers(config Config, encoder zapcore.Encoder, zapLevel zapcore.Level) (map[LogLevel]*zap.Logger, map[string]zapcore.WriteSyncer, error) {
	levelLoggers := make(map[LogLevel]*zap.Logger)
	syncers := make(map[string]zapcore.WriteSyncer)

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

	for level, info := range levelFiles {
		rotator, err := rotatelogs.New(
			filepath.Join(config.LogDir, info.fileName),
			rotatelogs.WithMaxAge(config.MaxAge),
			rotatelogs.WithRotationTime(config.RotationTime),
			rotatelogs.WithLinkName(filepath.Join(config.LogDir, info.fileName[:len(info.fileName)-len("_%Y%m%d.log")]+".log")),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize rotatelogs for %s: %w", info.fileName, err)
		}

		syncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(rotator), zapcore.AddSync(os.Stdout))
		syncers[fmt.Sprintf("level_%d", level)] = syncer

		levelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl == info.zapLevel && lvl >= zapLevel
		})

		core := zapcore.NewCore(encoder, syncer, levelEnabler)
		zapLogger := zap.New(core,
			zap.AddCaller(),
			zap.AddCallerSkip(1+config.CallerSkip),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
		levelLoggers[level] = zapLogger
	}

	return levelLoggers, syncers, nil
}

// 创建HTTP访问日志器 - 修复版本
func createHTTPLogger(config Config, encoder zapcore.Encoder, zapLevel zapcore.Level) (*zap.Logger, zapcore.WriteSyncer, error) {
	accessRotator, err := rotatelogs.New(
		filepath.Join(config.LogDir, "access_%Y%m%d.log"),
		rotatelogs.WithMaxAge(config.MaxAge),
		rotatelogs.WithRotationTime(config.RotationTime),
		rotatelogs.WithLinkName(filepath.Join(config.LogDir, "access.log")),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create access log rotator: %w", err)
	}

	accessSyncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(accessRotator), zapcore.AddSync(os.Stdout))
	accessCore := zapcore.NewCore(encoder, accessSyncer, zapLevel)
	logger := zap.New(accessCore,
		zap.AddCaller(),
		zap.AddCallerSkip(1+config.CallerSkip),
	)

	return logger, accessSyncer, nil
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
	if config.CallerSkip == 0 {
		config.CallerSkip = 1
	}

	// 确保日志目录存在
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// 获取日志级别
	zapLevel := getZapLevel(config.Level)

	// 创建编码器
	encoder := createEncoderWithSimpleStack(config.JSONFormat, config.CallerSkip)

	// 创建级别日志器
	levelLoggers, syncers, err := createLevelLoggers(config, encoder, zapLevel)
	if err != nil {
		return err
	}

	// 创建HTTP访问日志器
	accessLogger, accessSyncer, err := createHTTPLogger(config, encoder, zapLevel)
	if err != nil {
		return fmt.Errorf("failed to create HTTP logger: %w", err)
	}
	syncers["access"] = accessSyncer

	// 创建SQL日志器
	sqlLogger, sqlSyncer, err := createSQLLogger(config, encoder, zapLevel)
	if err != nil {
		return fmt.Errorf("failed to create SQL logger: %w", err)
	}
	syncers["sql"] = sqlSyncer

	globalLogger = &Logger{
		config:       config,
		syncers:      syncers,
		levelLoggers: levelLoggers,
		accessLogger: accessLogger,
		sqlLogger:    sqlLogger,
	}

	return nil
}

// 创建SQL日志器
func createSQLLogger(config Config, encoder zapcore.Encoder, zapLevel zapcore.Level) (*zap.Logger, zapcore.WriteSyncer, error) {
	sqlRotator, err := rotatelogs.New(
		filepath.Join(config.LogDir, "sql_%Y%m%d.log"),
		rotatelogs.WithMaxAge(config.MaxAge),
		rotatelogs.WithRotationTime(config.RotationTime),
		rotatelogs.WithLinkName(filepath.Join(config.LogDir, "sql.log")),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create sql log rotator: %w", err)
	}

	sqlSyncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(sqlRotator), zapcore.AddSync(os.Stdout))
	sqlCore := zapcore.NewCore(encoder, sqlSyncer, zapLevel)
	sqlLogger := zap.New(sqlCore,
		zap.AddCaller(),
		zap.AddCallerSkip(1+config.CallerSkip),
	)

	return sqlLogger, sqlSyncer, nil
}

// 获取简化的caller信息
func getSimplifiedCaller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0"
	}

	_, fileName := filepath.Split(file)

	// 获取函数名
	funcName := runtime.FuncForPC(pc).Name()
	if idx := strings.LastIndex(funcName, "/"); idx != -1 {
		funcName = funcName[idx+1:]
	}
	if idx := strings.Index(funcName, "."); idx != -1 {
		funcName = funcName[idx+1:]
	}

	return fmt.Sprintf("%s:%d:%s", fileName, line, funcName)
}

// Debug 记录Debug级别日志，带上下文
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[DebugLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	fields = append(fields, zap.String("caller", getSimplifiedCaller(2)))
	globalLogger.levelLoggers[DebugLevel].Debug(msg, fields...)
}

// Info 记录Info级别日志，带上下文
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[InfoLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	fields = append(fields, zap.String("caller", getSimplifiedCaller(2)))
	globalLogger.levelLoggers[InfoLevel].Info(msg, fields...)
}

// Warn 记录Warn级别日志，带上下文
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[WarnLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	fields = append(fields, zap.String("caller", getSimplifiedCaller(2)))
	globalLogger.levelLoggers[WarnLevel].Warn(msg, fields...)
}

// Error 记录Error级别日志，带上下文
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[ErrorLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	fields = append(fields,
		zap.String("caller", getSimplifiedCaller(2)),
		zap.String("stacktrace", getSimplifiedStack(3)),
	)
	globalLogger.levelLoggers[ErrorLevel].Error(msg, fields...)
}

// Fatal 记录Fatal级别日志，带上下文
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	if globalLogger == nil || globalLogger.levelLoggers[FatalLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		os.Exit(1)
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	fields = append(fields,
		zap.String("caller", getSimplifiedCaller(2)),
		zap.String("stacktrace", getSimplifiedStack(3)),
	)
	globalLogger.levelLoggers[FatalLevel].Fatal(msg, fields...)
}

// LogCustomError 记录自定义错误，带上下文
func LogCustomError(ctx context.Context, customErr *CustomError) {
	if globalLogger == nil || globalLogger.levelLoggers[ErrorLevel] == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return
	}
	fields := []zap.Field{
		zap.String("error_code", customErr.Code),
		zap.String("caller", getSimplifiedCaller(2)),
		zap.String("stacktrace", getSimplifiedStack(3)),
	}
	if traceID := trace.GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	for k, v := range customErr.Fields {
		fields = append(fields, zap.Any(k, v))
	}
	globalLogger.levelLoggers[ErrorLevel].Error(customErr.Message, fields...)
}

// GetHTTPLogger 获取HTTP日志器
func GetHTTPLogger() *zap.Logger {
	if globalLogger == nil {
		return nil
	}
	return globalLogger.accessLogger
}

// GetSQLLogger 获取SQL日志器
func GetSQLLogger() *zap.Logger {
	if globalLogger == nil {
		return nil
	}
	return globalLogger.sqlLogger
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
	if globalLogger.accessLogger != nil {
		if err := globalLogger.accessLogger.Sync(); err != nil {
			lastErr = fmt.Errorf("failed to sync access logger: %w", err)
		}
	}
	if globalLogger.sqlLogger != nil {
		if err := globalLogger.sqlLogger.Sync(); err != nil {
			lastErr = fmt.Errorf("failed to sync sql logger: %w", err)
		}
	}
	return lastErr
}

// Close 关闭日志器
func Close() error {
	return Sync()
}
