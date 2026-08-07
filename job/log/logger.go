package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/config"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const ServiceName = "usergroup-job"

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
		With(zap.String("service", serviceName()), zap.String("env", envName())).
		Sugar()
}

// WithContext returns a JSON logger always enriched with trace_id / span_id.
func WithContext(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		ctx = context.Background()
	}
	traceID, spanID := TraceIDs(ctx)
	return Logger.With("trace_id", traceID, "span_id", spanID)
}

// TraceIDs extracts W3C trace/span ids from ctx.
func TraceIDs(ctx context.Context) (string, string) {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return "", ""
	}
	return sc.TraceID().String(), sc.SpanID().String()
}

func getLogLevel() zapcore.Level {
	level := zapcore.InfoLevel
	if config.Config != nil && config.Config.LogConfig != nil && config.Config.LogConfig.Level != "" {
		if err := level.UnmarshalText([]byte(config.Config.LogConfig.Level)); err != nil {
			level = zapcore.InfoLevel
		}
	}
	return level
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "msg"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {
	if config.Config == nil || config.Config.LogConfig == nil || config.Config.LogConfig.FilePath == "" {
		return zapcore.AddSync(os.Stdout)
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get current working directory: %v", err))
	}
	logPath := filepath.Join(cwd, config.Config.LogConfig.FilePath)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return zapcore.AddSync(os.Stdout)
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return zapcore.AddSync(os.Stdout)
	}
	return zapcore.AddSync(f)
}

func envName() string {
	for _, key := range []string{"APP_ENV", "RAILWAY_ENVIRONMENT", "GO_ENV"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return "local"
}

func serviceName() string {
	if v := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); v != "" {
		return v
	}
	return ServiceName
}
