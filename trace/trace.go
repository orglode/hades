package trace

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log/slog"
)

// contextKey 定义上下文中的traceID键
type contextKey string

const traceIDKey contextKey = "traceID"

// GetTraceID 从上下文中提取traceID
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		return traceID
	}
	return ""
}

// TraceIDMiddleware 生成并注入 TraceId 的中间件
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成唯一的 TraceId
		traceID := uuid.New().String()

		// 将 TraceId 存入 Gin 上下文
		c.Set("trace_id", traceID)

		// 将 TraceId 存入请求的 Context，方便下游使用
		ctx := context.WithValue(c.Request.Context(), traceIDKey, traceID)
		c.Request = c.Request.WithContext(ctx)

		// 记录请求日志
		slog.Info("Handling request", "path", c.Request.URL.Path, "trace_id", traceID)

		// 继续处理请求
		c.Next()
	}
}
