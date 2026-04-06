package logger_v2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GinLogger 返回Gin日志中间件
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 读取请求体
		var requestBody interface{}
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil && len(bodyBytes) > 0 {
				// 尝试解析为JSON
				var bodyMap map[string]interface{}
				if json.Unmarshal(bodyBytes, &bodyMap) == nil {
					requestBody = bodyMap
				} else {
					// 如果不是JSON，则保存为字符串
					requestBody = string(bodyBytes)
				}
				// 重新设置请求体
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// 处理请求
		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		// 设置为东八区北京时间
		tz, _ := time.LoadLocation("Asia/Shanghai")

		// 收集请求头
		headers := make(map[string]string)
		importantHeaders := []string{
			"Content-Type", "Authorization", "User-Agent",
			"X-Request-ID", "X-Forwarded-For", "Referer",
		}

		for _, headerName := range importantHeaders {
			if val := c.GetHeader(headerName); val != "" {
				headers[headerName] = val
			}
		}

		// 获取traceID
		var traceID string
		if val, exists := c.Get("trace_id"); exists {
			if tid, ok := val.(string); ok {
				traceID = tid
			}
		}
		// 创建请求信息
		reqInfo := GinRequestInfo{
			Method:       c.Request.Method,
			Path:         path,
			Query:        query,
			Status:       c.Writer.Status(),
			IP:           c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			Latency:      latency.String(),
			RequestBody:  requestBody,
			Headers:      headers,
			ResponseSize: c.Writer.Size(),
			Timestamp:    end.In(tz).Format("2006-01-02 15:04:05"),
			TraceID:      traceID,
		}

		// 根据状态码选择日志级别
		status := c.Writer.Status()
		logFields := []zap.Field{
			zap.Any("request", reqInfo),
			zap.String("caller", getSimplifiedCaller(2)),
		}

		// 记录日志
		if globalLogger != nil && globalLogger.accessLogger != nil {
			switch {
			case status >= 500:
				globalLogger.accessLogger.Error("Server Error", logFields...)
			case status >= 400:
				globalLogger.accessLogger.Warn("Client Error", logFields...)
			default:
				globalLogger.accessLogger.Info("Request", logFields...)
			}
		} else {
			// 如果日志器未初始化，输出到控制台
			logStr, _ := json.Marshal(reqInfo)
			fmt.Printf("[GIN] %s %s %d %s - %s\n",
				reqInfo.Method, reqInfo.Path, reqInfo.Status,
				reqInfo.Latency, string(logStr))
		}
	}
}
