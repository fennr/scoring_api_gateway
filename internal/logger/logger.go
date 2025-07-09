package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level string, json bool) (*zap.Logger, error) {
	var config zap.Config

	if json {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	parsedLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		parsedLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(parsedLevel)

	return config.Build()
}
