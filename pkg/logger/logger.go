package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/agent-collector/config"
)

type Logger = zap.Logger

var (
	baseLogger    *zap.Logger
	defaultFields = struct {
		Collector string
	}{}
	loggerInitOnce    sync.Once
	loggerInitialized bool
	mu                sync.RWMutex
)

func Init(cfg config.ZapLogConfig) error {
	var err error
	loggerInitOnce.Do(func() {
		level := zapcore.InfoLevel
		switch strings.ToLower(cfg.Level) {
		case "dbg", "debug":
			level = zapcore.DebugLevel
		case "inf", "info":
			level = zapcore.InfoLevel
		case "war", "warn":
			level = zapcore.WarnLevel
		case "err", "error":
			level = zapcore.ErrorLevel
		case "pan", "panic":
			level = zapcore.PanicLevel
		case "fat", "fatal":
			level = zapcore.FatalLevel
		}

		if err = os.MkdirAll(cfg.Path, 0755); err != nil {
			return
		}

		writer, wErr := rotatelogs.New(
			filepath.Join(cfg.Path, "agent-%Y%m%d.log"),
			rotatelogs.WithMaxAge(7*24*time.Hour),
			rotatelogs.WithRotationTime(24*time.Hour),
			rotatelogs.WithRotationSize(100*1024*1024),
		)
		if wErr != nil {
			err = wErr
			return
		}

		// 控制台彩色时间
		customTimeEncoderConsole := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(fmt.Sprintf("\033[34m%s\033[0m", t.Format("2006-01-02 15:04:05.000 -07:00")))
		}

		// JSON 日志纯文本时间
		customTimeEncoderJSON := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000 -07:00"))
		}

		coloredLevelEncoder := func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			var levelStr string
			switch level {
			case zapcore.DebugLevel:
				levelStr = "\033[36mDEBUG\033[0m"
			case zapcore.InfoLevel:
				levelStr = "\033[32mINFO \033[0m"
			case zapcore.WarnLevel:
				levelStr = "\033[33mWARN \033[0m"
			case zapcore.ErrorLevel:
				levelStr = "\033[31mERROR\033[0m"
			case zapcore.DPanicLevel:
				levelStr = "\033[35mDPANIC\033[0m"
			case zapcore.PanicLevel:
				levelStr = "\033[35mPANIC\033[0m"
			case zapcore.FatalLevel:
				levelStr = "\033[35mFATAL\033[0m"
			default:
				levelStr = "UNK  "
			}
			enc.AppendString(levelStr)
		}

		consoleEncoderCfg := zap.NewDevelopmentEncoderConfig()
		consoleEncoderCfg.ConsoleSeparator = " "
		consoleEncoderCfg.EncodeLevel = coloredLevelEncoder
		consoleEncoderCfg.EncodeTime = customTimeEncoderConsole

		// Caller 两级路径
		consoleEncoderCfg.EncodeCaller = func(c zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			rel := filepath.Join(filepath.Base(filepath.Dir(c.File)), filepath.Base(c.File))
			enc.AppendString(fmt.Sprintf("%s:%d", rel, c.Line))
		}

		consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderCfg)

		jsonCfg := zap.NewProductionEncoderConfig()
		jsonCfg.TimeKey = "timestamp"
		jsonCfg.EncodeTime = customTimeEncoderJSON
		jsonCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		jsonEncoder := zapcore.NewJSONEncoder(jsonCfg)

		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
			zapcore.NewCore(jsonEncoder, zapcore.AddSync(writer), level),
		)

		baseLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
		loggerInitialized = true
	})
	return err
}

func SetDefaultCollector(collector string) {
	mu.Lock()
	defer mu.Unlock()
	defaultFields.Collector = collector
}

func GetDefaultCollector() string {
	mu.RLock()
	defer mu.RUnlock()
	return defaultFields.Collector
}

func getDefaultFields(collectorOverride ...string) []zapcore.Field {
	mu.RLock()
	collector := defaultFields.Collector
	mu.RUnlock()

	if len(collectorOverride) > 0 && collectorOverride[0] != "" {
		collector = collectorOverride[0]
	}

	return []zapcore.Field{
		zap.String("collector", collector),
		zap.String("goid", getGID()),
	}
}

func getGID() string {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))
	if len(idField) > 0 {
		if id, err := strconv.Atoi(idField[0]); err == nil {
			return strconv.Itoa(id)
		}
	}
	return "0"
}

func log(level zapcore.Level, msg string, collectorOverride string, fields ...zapcore.Field) {
	if !loggerInitialized {
		panic("logger not initialized: call logger.Init() first")
	}

	mergedFields := getDefaultFields(collectorOverride)
	msgWithFields := fmt.Sprintf("collector=%s goid=%s %s", mergedFields[0].String, mergedFields[1].String, msg)

	loggerWithFields := baseLogger.WithOptions(zap.AddCallerSkip(1)).With(fields...)

	switch level {
	case zap.DebugLevel:
		loggerWithFields.Debug(msgWithFields)
	case zap.InfoLevel:
		loggerWithFields.Info(msgWithFields)
	case zap.WarnLevel:
		loggerWithFields.Warn(msgWithFields)
	case zap.ErrorLevel:
		loggerWithFields.Error(msgWithFields)
	case zap.PanicLevel:
		loggerWithFields.Panic(msgWithFields)
	case zap.FatalLevel:
		loggerWithFields.Fatal(msgWithFields)
	}
}

func Debug(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.DebugLevel, msg, collectorOverride, fields...)
}
func Info(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.InfoLevel, msg, collectorOverride, fields...)
}
func Warn(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.WarnLevel, msg, collectorOverride, fields...)
}
func Error(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.ErrorLevel, msg, collectorOverride, fields...)
}
func Panic(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.PanicLevel, msg, collectorOverride, fields...)
}
func Fatal(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.FatalLevel, msg, collectorOverride, fields...)
}

func Sync() error {
	if !loggerInitialized {
		return nil
	}
	return baseLogger.Sync()
}

func GetLogger() *zap.Logger {
	if !loggerInitialized {
		panic("logger not initialized: call logger.Init() first")
	}
	return baseLogger
}
