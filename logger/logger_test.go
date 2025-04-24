package logger

import (
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	log, err := NewLogger("", Info)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Sync()
	// 2. 设置全局panic捕获
	defer log.Recover()
	log.Info("Hello World")
	log.Info("这是一条测试日志", zap.String("启动时间", time.Now().Format(time.RFC3339)))

}
