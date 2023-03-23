package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var I *zap.Logger

func init() {
	atom := zap.NewAtomicLevel()
	config := zap.NewProductionEncoderConfig() // or zap.NewDevelopmentConfig() or any other zap.Config
	config.TimeKey = "timestamp"
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncodeCaller = zapcore.FullCallerEncoder
	I = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(config),
		zapcore.Lock(os.Stdout),
		atom,
	))
	atom.SetLevel(zap.DebugLevel)
}

func MapStrings2Fields(m map[string]string) []zap.Field {
	fields := make([]zap.Field, 0, len(m))
	for k, v := range m {
		fields = append(fields, zap.String(k, v))
	}
	return fields
}
