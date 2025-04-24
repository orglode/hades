package logger

import (
	"fmt"
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	config := Config{
		LogDir:       "./logs",
		MaxAge:       30 * 24 * time.Hour,
		RotationTime: 24 * time.Hour,
		Level:        "debug",
		JSONFormat:   true,
	}
	if err := InitLogger(config); err != nil {
		panic(err)
	}
	defer Close()

	// 测试日志级别
	Debug("This is a debug message", zap.String("key", "debug"))
	Info("This is an info message", zap.Int("count", 42))
	Warn("This is a warn message", zap.String("context", "warning"))
	Error("This is an error message", zap.Error(fmt.Errorf("something went wrong")))

	// 测试自定义错误
	customErr := NewCustomError("ERR001", "test error", map[string]interface{}{"retry": 3})
	LogCustomError(customErr)

}
