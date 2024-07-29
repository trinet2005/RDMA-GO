package RDMAGO

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var sugarLogger *zap.SugaredLogger

func init() {
	InitLog(false)
}

func InitLog(debugMode bool) {
	var cfg zap.Config
	if debugMode {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel) // 只记录Info级别及以上的日志
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugarLogger = logger.Sugar()
}

func LogInfo(message string) {
	sugarLogger.Infof("zap Info: %s", message)
}

func LogDebug(message string) {
	sugarLogger.Debugf("zap Debug: %s", message)
}

func LogError(message string, err error) {
	sugarLogger.Errorf("zap Error: %s, %v", message, err)
}
