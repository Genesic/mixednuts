package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"strings"
	"time"
)

type RequestID string

const RequestIDKey RequestID = "request_id"

type contextKey string

const loggerKey = contextKey("logger")

const (
	timestamp = "ts"
	// severity key "severity" will be captured by Google Logging to report the level
	// of the log entry.
	severity = "severity"
	logger   = "logger"
	caller   = "caller"
	message  = "msg"

	levelDebug     = "DEBUG"
	levelInfo      = "INFO"
	levelWarning   = "WARNING"
	levelError     = "ERROR"
	levelCritical  = "CRITICAL"
	levelAlert     = "ALERT"
	levelEmergency = "EMERGENCY"
)

var (
	encoderConfig = zapcore.EncoderConfig{
		TimeKey:        timestamp,
		LevelKey:       severity,
		NameKey:        logger,
		CallerKey:      caller,
		MessageKey:     message,
		StacktraceKey:  zapcore.OmitKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     timeEncoder(),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	DefaultLogLevel  = levelInfo
	DefaultIsDevMode = false
)

// SetDefaultConfig initializes logging module internal state
func SetDefaultConfig(logLevel string, isDevMode bool) {
	DefaultLogLevel = logLevel
	DefaultIsDevMode = isDevMode
}

// WithLogger creates a new context with the provided logger attached.
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger stored in the context. If no such logger
// exists, a default logger is returned.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	sugaredLogger := fromContext(ctx)
	requestID, _ := ctx.Value(RequestIDKey).(string)
	if requestID != "" {
		sugaredLogger = sugaredLogger.With(string(RequestIDKey), requestID)
	}
	return sugaredLogger
}

func fromContext(ctx context.Context) *zap.SugaredLogger {
	if sugaredLogger, ok := ctx.Value(loggerKey).(*zap.SugaredLogger); ok {
		return sugaredLogger
	}
	return NewDefaultLogger()
}

// NewLogger returns the logger used across the codebase. The argument "level"
// specifies the minimum level the logger will write. A zero string means the
// logger will print debug entries in dev mode; otherwise it prints info entries.
// See https://pkg.go.dev/go.uber.org/zap#example-package-AdvancedConfiguration
// for how the zap logger is configured.
func NewLogger(level string, devMode bool) *zap.SugaredLogger {
	if level == "" {
		if devMode {
			level = levelDebug
		} else {
			level = levelInfo
		}
	}

	zapLevel := levelToZapLevel(level)
	normalLevel := zap.LevelEnablerFunc(func(lv zapcore.Level) bool {
		if stdErrLv(lv) {
			return false
		}
		return lv >= zapLevel
	})

	errorFatalLevel := zap.LevelEnablerFunc(stdErrLv)

	stdoutWriteSyncer := zapcore.Lock(os.Stdout)
	stderrWriterSyncer := zapcore.Lock(os.Stderr)

	consoleEncoder := zapcore.NewJSONEncoder(encoderConfig)

	var opts = []zap.Option{
		zap.AddCaller(),
		zap.ErrorOutput(stderrWriterSyncer),
		zap.Fields(),
	}
	if devMode {
		opts = append(opts,
			zap.Development(),
			zap.Fields(
				zap.Bool("dev", devMode),
			),
		)
	}

	core := zapcore.NewTee(
		zapcore.NewCore(
			consoleEncoder,
			stdoutWriteSyncer,
			normalLevel,
		),
		zapcore.NewCore(
			consoleEncoder,
			stderrWriterSyncer,
			errorFatalLevel,
		),
	)

	logger := zap.New(core, opts...)
	return logger.Sugar()
}

func stdErrLv(lv zapcore.Level) bool {
	return lv == zapcore.ErrorLevel || lv == zapcore.FatalLevel
}

func NewDefaultLogger() *zap.SugaredLogger {
	return NewLogger(DefaultLogLevel, DefaultIsDevMode)
}

func levelToZapLevel(s string) zapcore.Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case levelDebug:
		return zapcore.DebugLevel
	case levelInfo:
		return zapcore.InfoLevel
	case levelWarning:
		return zapcore.WarnLevel
	case levelError:
		return zapcore.ErrorLevel
	case levelCritical:
		return zapcore.DPanicLevel
	case levelAlert:
		return zapcore.PanicLevel
	case levelEmergency:
		return zapcore.FatalLevel
	}

	return zapcore.WarnLevel
}

// timeEncoder encodes the time as RFC3339 nano with only 4 digits
func timeEncoder() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02T15:04:05.9999Z07:00"))
	}
}

// safeJSONString returns a valid JSON string.
func safeJSONString(v interface{}) string {
	if err, ok := v.(error); ok {
		v = err.Error()
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%#+v", v)
	}
	// Trim the beginning and trailing " character
	return string(b[1 : len(b)-1])
}

func FinalFatal(v interface{}) {
	log.SetFlags(0)
	var s = fmt.Sprintf(`{"%s": "%s", "msg": "%v"`,
		severity,
		levelCritical,
		safeJSONString(v))

	s += "}"
	log.Fatalf(s)
}
