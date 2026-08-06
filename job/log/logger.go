package log

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger = zap.NewNop().Sugar()

func InitLogger() {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, getLogLevel())
	if config.Config != nil && config.Config.LogConfig != nil && config.Config.LogConfig.FilePath != "" {
		consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), getLogLevel())
		core = zapcore.NewTee(core, consoleCore)
	}
	Logger = zap.New(core, zap.AddCaller()).
		With(zap.String("service", "usergroup-job"), zap.String("env", envName())).
		Sugar()
}

func getLogLevel() zapcore.Level {
	level := zapcore.InfoLevel
	if config.Config != nil && config.Config.LogConfig != nil && config.Config.LogConfig.Level != "" {
		_ = level.UnmarshalText([]byte(config.Config.LogConfig.Level))
	}
	return level
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {
	path := ""
	if config.Config != nil && config.Config.LogConfig != nil {
		path = config.Config.LogConfig.FilePath
	}
	if path == "" {
		return zapcore.AddSync(os.Stdout)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return zapcore.AddSync(os.Stdout)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return zapcore.AddSync(os.Stdout)
	}
	return zapcore.AddSync(f)
}

func envName() string {
	if v := strings.TrimSpace(os.Getenv("APP_ENV")); v != "" {
		return v
	}
	return "local"
}
