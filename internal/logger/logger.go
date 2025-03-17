package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(logLevel string) (*zap.SugaredLogger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		level = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	l, err := config.Build()
	if err != nil {
		return nil, err
	}

	return l.Sugar(), nil
}
