package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *zap.Logger

// Init initialises the global logger.
// level: "debug" | "info" | "warn" | "error"
// format: "json" | "console"
// output: "stdout" | "stderr" | file path
func Init(level, format, output string) error {
	zapLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		zapLevel = zapcore.InfoLevel
	}

	var encoder zapcore.Encoder
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	if format == "console" {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	var writeSyncer zapcore.WriteSyncer
	switch output {
	case "stderr":
		writeSyncer = zapcore.AddSync(os.Stderr)
	case "stdout", "":
		writeSyncer = zapcore.AddSync(os.Stdout)
	default:
		f, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writeSyncer = zapcore.AddSync(f)
	}

	core := zapcore.NewCore(encoder, writeSyncer, zapLevel)
	globalLogger = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	return nil
}

// Get returns the global logger. Initialises a development logger if Init has not been called.
func Get() *zap.Logger {
	if globalLogger == nil {
		l, _ := zap.NewDevelopment()
		globalLogger = l
	}
	return globalLogger
}

func Info(msg string, fields ...zap.Field)  { Get().Info(msg, fields...) }
func Error(msg string, fields ...zap.Field) { Get().Error(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { Get().Warn(msg, fields...) }
func Debug(msg string, fields ...zap.Field) { Get().Debug(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { Get().Fatal(msg, fields...) }

// With returns a child logger with the given fields attached.
func With(fields ...zap.Field) *zap.Logger { return Get().With(fields...) }

// Sync flushes any buffered log entries.
func Sync() { _ = Get().Sync() }
