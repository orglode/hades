package logging

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

func initCoreEncoder(initLevel string) zapcore.Core {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.MessageKey = "message"
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	//转换时间
	encoderConfig.EncodeTime = timeLayout
	// level 大写
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	//初始化
	level := zap.NewAtomicLevelAt(zap.DebugLevel)

	writeSyncer, _ := os.Create(logFileName(initLevel))
	//初始化core
	encoder := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.Lock(writeSyncer), level)
	return encoder
}

func timeLayout(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func logFileName(initLog string) string {
	ok, dirInfo := initFilePath()
	if !ok {
		return ""
	}
	newFile := dirInfo + initLog + "_" + time.Now().Format("20060102") + ".log"
	return newFile
}

func initLeve() map[string]zapcore.Level {
	levelMap := map[string]zapcore.Level{
		"error": zapcore.ErrorLevel,
		"info":  zapcore.InfoLevel,
		"debug": zapcore.DebugLevel,
		"panic": zapcore.PanicLevel,
	}
	return levelMap
}

func initFilePath() (bool, string) {
	filePath := "./logs/"
	if ok, _ := pathExists(filePath); ok {
		return true, filePath
	}
	//创建文件
	err := os.Mkdir(filePath, os.ModePerm)
	if err != nil {
		fmt.Printf("创建目录异常 -> %v \n", err)
		return false, ""
	}
	return true, filePath
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
