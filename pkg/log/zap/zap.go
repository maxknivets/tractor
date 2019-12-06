package zap

import "go.uber.org/zap"

func NewLogger() *zap.SugaredLogger {
	logger, _ := zap.NewDevelopment()
	return logger.Sugar()
}
