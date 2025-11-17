package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/agent-collector/config"
)

// Logger 别名（兼容原有代码）
type Logger = zap.Logger

// 全局变量：存储基础日志实例和默认配置（线程安全）
var (
	baseLogger        *zap.Logger
	defaultCollector  string       // 默认collector名称
	defaultGid        string       // 默认gid
	loggerInitOnce    sync.Once    // 确保初始化只执行一次
	loggerInitialized bool         // 标记是否已初始化
	mu                sync.RWMutex // 保护默认配置的读写
)

// Init 初始化全局日志（程序启动时调用一次即可）
// 替代原有的 NewZapLogger 直接使用，初始化后可直接调用包级快捷函数
func Init(cfg config.ZapLogConfig) error {
	var err error
	loggerInitOnce.Do(func() {
		// 日志级别解析
		level := zapcore.InfoLevel
		switch cfg.Level {
		case "DBG", "Debug", "debug":
			level = zapcore.DebugLevel
		case "INF", "Info", "info":
			level = zapcore.InfoLevel
		case "WAR", "Warn", "warn":
			level = zapcore.WarnLevel
		case "ERR", "Error", "error":
			level = zapcore.ErrorLevel
		case "PAN", "Panic", "panic":
			level = zapcore.PanicLevel
		case "FAT", "Fatal", "fatal":
			level = zapcore.FatalLevel
		}

		// 创建日志目录
		if err = os.MkdirAll(cfg.Path, 0755); err != nil {
			return
		}

		// 文件轮转配置
		var writer *rotatelogs.RotateLogs
		writer, err = rotatelogs.New(
			filepath.Join(cfg.Path, "agent-%Y%m%d.log"),
			rotatelogs.WithMaxAge(7*24*time.Hour),
			rotatelogs.WithRotationTime(24*time.Hour),
			rotatelogs.WithRotationSize(100*1024*1024),
		)
		if err != nil {
			return
		}

		// 自定义时间编码并且添加颜色
		customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			// 蓝色时间，\033[34m 开始蓝色，\033[0m 重置颜色
			//如 31=红色，32=绿色，33=黄色，34=蓝色
			enc.AppendString(fmt.Sprintf("\033[34m%s\033[0m", t.Format("2006-01-02 15:04:05.000 -07:00")))
		}

		// ------------------ 修改: 对齐并彩色化日志级别 ------------------
		alignedColorLevelEncoder := func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			var levelStr string
			switch level {
			case zapcore.DebugLevel:
				levelStr = "\033[36mDEBUG\033[0m" // 青色
			case zapcore.InfoLevel:
				levelStr = "\033[32mINFO \033[0m" // 绿色，注意补空格对齐
			case zapcore.WarnLevel:
				levelStr = "\033[33mWARN \033[0m" // 黄色
			case zapcore.ErrorLevel:
				levelStr = "\033[31mERROR\033[0m" // 红色
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
		// ----------------------------------------------------------

		// 控制台编码器（修改时间 + 对齐级别）
		consoleEncoderCfg := zap.NewDevelopmentEncoderConfig()
		consoleEncoderCfg.ConsoleSeparator = " "
		consoleEncoderCfg.EncodeLevel = alignedColorLevelEncoder // 修改: 对齐 + 彩色
		consoleEncoderCfg.EncodeTime = customTimeEncoder         // 修改: 自定义时间格式
		consoleEncoderCfg.EncodeCaller = zapcore.ShortCallerEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderCfg)

		// JSON文件编码器（只修改时间格式）
		jsonEncoderCfg := zap.NewProductionEncoderConfig()
		jsonEncoderCfg.TimeKey = "timestamp"
		jsonEncoderCfg.EncodeTime = customTimeEncoder // 修改: 自定义时间格式
		jsonEncoderCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		jsonEncoder := zapcore.NewJSONEncoder(jsonEncoderCfg)

		// 双输出核心（控制台+文件）
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
			zapcore.NewCore(jsonEncoder, zapcore.AddSync(writer), level),
		)

		// 构建基础logger
		baseLogger = zap.New(
			core,
			zap.AddCaller(),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
		loggerInitialized = true
	})
	return err
}

// SetDefaultCollector 设置全局默认collector名称（可选，后续调用可省略collector参数）
func SetDefaultCollector(collector string) {
	mu.Lock()
	defer mu.Unlock()
	defaultCollector = collector
}

// SetDefaultGid 设置全局默认gid（可选，后续调用可省略gid参数）
func SetDefaultGid(gid string) {
	mu.Lock()
	defer mu.Unlock()
	defaultGid = gid
}

// --------------- 包级快捷日志函数（无需创建实例，直接调用）---------------
// Debug 调试级别日志：支持 (collector, gid, msg) 或 ( "", "", msg )（使用默认值）
func Debug(collector, gid, msg string, fields ...zapcore.Field) {
	log(zap.DebugLevel, collector, gid, msg, fields...)
}

// Info 信息级别日志：支持 (collector, gid, msg) 或 ( "", "", msg )（使用默认值）
func Info(collector, gid, msg string, fields ...zapcore.Field) {
	log(zap.InfoLevel, collector, gid, msg, fields...)
}

// Warn 警告级别日志：支持 (collector, gid, msg) 或 ( "", "", msg )（使用默认值）
func Warn(collector, gid, msg string, fields ...zapcore.Field) {
	log(zap.WarnLevel, collector, gid, msg, fields...)
}

// Error 错误级别日志：支持 (collector, gid, msg) 或 ( "", "", msg )（使用默认值）
func Error(collector, gid, msg string, fields ...zapcore.Field) {
	log(zap.ErrorLevel, collector, gid, msg, fields...)
}

// Panic 恐慌级别日志：支持 (collector, gid, msg) 或 ( "", "", msg )（触发panic）
func Panic(collector, gid, msg string, fields ...zapcore.Field) {
	log(zap.PanicLevel, collector, gid, msg, fields...)
}

// Fatal 致命级别日志：支持 (collector, gid, msg) 或 ( "", "", msg )（终止程序）
func Fatal(collector, gid, msg string, fields ...zapcore.Field) {
	log(zap.FatalLevel, collector, gid, msg, fields...)
}

// Sync 刷新日志缓冲区（程序退出前调用一次即可）
func Sync() error {
	if !loggerInitialized {
		return nil
	}
	return baseLogger.Sync()
}

// log 核心日志输出函数（内部使用，处理级别、collector、gid的合并）
func log(level zapcore.Level, collector, gid, msg string, fields ...zapcore.Field) {
	// 检查日志是否已初始化（未初始化则panic提示）
	if !loggerInitialized {
		panic("logger not initialized: call logger.Init() first")
	}

	// 读取默认配置（带读锁，线程安全）
	mu.RLock()
	defer mu.RUnlock()

	// 优先使用传入的collector/gid，为空则使用默认值
	usedCollector := collector
	if usedCollector == "" {
		usedCollector = defaultCollector
	}
	usedGid := gid
	if usedGid == "" {
		usedGid = defaultGid
	}

	// 合并固定字段（collector + gid）和额外字段
	mergedFields := []zapcore.Field{
		zap.String("collector", usedCollector),
		zap.String("goid", usedGid),
	}
	mergedFields = append(mergedFields, fields...)

	// 根据级别输出日志
	switch level {
	case zap.DebugLevel:
		baseLogger.Debug(msg, mergedFields...)
	case zap.InfoLevel:
		baseLogger.Info(msg, mergedFields...)
	case zap.WarnLevel:
		baseLogger.Warn(msg, mergedFields...)
	case zap.ErrorLevel:
		baseLogger.Error(msg, mergedFields...)
	case zap.PanicLevel:
		baseLogger.Panic(msg, mergedFields...)
	case zap.FatalLevel:
		baseLogger.Fatal(msg, mergedFields...)
	}
}

// GetLogger 获取初始化后的全局 zap.Logger 实例，提供外部函数参数传递
// 注意：必须在 logger.Init() 调用成功后使用，否则返回nil并且panic
func GetLogger() *zap.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if !loggerInitialized {
		panic("logger not initialized: call logger.Init() first")
	}
	return baseLogger
}
