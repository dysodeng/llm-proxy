package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"llm-proxy/internal/config"
)

// New creates a zap logger that writes to both stdout and a rotating log file.
// Log file rotates daily via lumberjack, retaining logs for cfg.MaxAge days.
// If cfg.File is empty, only stdout is used.
func New(cfg config.LogConfig) (*zap.Logger, error) {
	level := parseLevel(cfg.Level)

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)

	stdoutSink := zapcore.AddSync(os.Stdout)
	stdoutCore := zapcore.NewCore(consoleEncoder, stdoutSink, level)

	var core zapcore.Core

	if cfg.File != "" {
		rotator := &lumberjack.Logger{
			Filename: cfg.File,
			MaxAge:   cfg.MaxAge,
			Compress: true,
		}
		fileSink := zapcore.AddSync(rotator)
		fileCore := zapcore.NewCore(jsonEncoder, fileSink, level)
		core = zapcore.NewTee(stdoutCore, fileCore)
	} else {
		core = stdoutCore
	}

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger, nil
}

// parseLevel converts a level string to a zapcore.Level, defaulting to InfoLevel.
func parseLevel(s string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
