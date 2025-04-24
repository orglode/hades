package logger

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"os"
	"time"
)

// GinMiddleware 返回Gin的日志中间件
func GinMiddleware() gin.HandlerFunc {
	if globalLogger == nil || globalLogger.accessLogger == nil {
		fmt.Fprintln(os.Stderr, "logger not initialized")
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		ctx := c.Request.Context()
		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
		}
		if traceID := getTraceID(ctx); traceID != "" {
			fields = append(fields, zap.String("traceID", traceID))
		}

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				// Gin错误日志写入error_*.log和终端
				Error(ctx, err.Error(), fields...)
			}
		} else {
			// 正常请求日志写入access_*.log和终端
			globalLogger.accessLogger.Ctx(ctx).Info("request processed", fields...)
		}
	}
}
