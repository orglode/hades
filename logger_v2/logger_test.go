// simple_test.go
package logger_v2

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"
)

// 简单模拟trace包
func getTraceID(ctx context.Context) string {
	return "test-trace-123"
}

func TestSimpleLog(t *testing.T) {
	// 1. 初始化日志
	config := Config{
		LogDir:     "./test_logs_simple",
		Level:      "debug",
		JSONFormat: true, // false 时是彩色控制台输出
		CallerSkip: 1,
	}

	InitLogger(config)
	defer Close()

	// 2. 模拟一个上下文
	ctx := context.Background()

	// 3. 打几条测试日志看看
	fmt.Println("=== 测试日志输出 ===")

	Debug(ctx, "这是一条Debug日志")
	Info(ctx, "这是一条Info日志")
	Warn(ctx, "这是一条Warn日志")
	Error(ctx, "这是一条Error日志", zap.String("error_detail", "文件不存在"))

	// 4. 自定义错误
	customErr := NewCustomError("ERR_001", "自定义错误测试",
		map[string]interface{}{
			"user_id": 1001,
			"action":  "login",
		})
	LogCustomError(ctx, customErr)

	fmt.Println("=== 测试完成，查看控制台输出和日志文件 ===")
}
