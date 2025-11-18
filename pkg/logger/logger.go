// logger包基于zap和file-rotatelogs实现高性能日志工具，支持以下核心特性：
// 1. 双输出目标：控制台彩色格式化输出 + 文件JSON格式持久化
// 2. 日志轮转：按天自动轮转日志文件，保留7天历史日志
// 3. 级别过滤：支持debug/info/warn/error/panic/fatal六级日志过滤
// 4. 默认字段：自动注入collector（可覆盖）和goroutine ID字段
// 5. 增强可读性：控制台输出带颜色区分（时间蓝/级别多色/调用者精简路径）
// 6. 线程安全：默认字段读写通过读写锁保护，支持并发场景
// 7. 调试友好：错误级别日志自动附加堆栈信息，调用者信息包含文件路径+行号
//
// 使用规范：
// 1. 程序启动时必须先调用Init()初始化，传入日志配置
// 2. 可通过SetDefaultCollector()设置全局默认collector字段
// 3. 日志输出优先使用包暴露的Debug/Info/Warn等方法，支持临时覆盖collector
// 4. 程序退出前建议调用Sync()确保日志缓冲区数据刷盘
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

// Logger 是zap.Logger的类型别名，简化外部包对日志核心类型的引用
type Logger = zap.Logger

var (
	// baseLogger 日志核心实例，由Init()函数初始化，所有日志输出均基于此实例
	baseLogger *zap.Logger
	// defaultFields 存储全局默认日志字段，目前包含collector（采集器标识）
	defaultFields = struct {
		Collector string // 采集器名称，用于日志分类筛选
	}{}
	// loggerInitOnce 确保日志初始化逻辑只执行一次（单例模式），避免重复初始化
	loggerInitOnce sync.Once
	// loggerInitialized 日志初始化状态标记，未初始化时调用日志方法会panic
	loggerInitialized bool
	// mu 读写锁，保护defaultFields的并发读写安全（多goroutine场景下安全设置/获取collector）
	mu sync.RWMutex
)

// Init 初始化日志系统，必须在使用任何日志方法前调用（建议程序启动时执行）
// 参数cfg：日志配置结构体，包含日志级别、存储路径等核心配置
// 返回值：初始化过程中产生的错误（如目录创建失败、日志轮转器初始化失败等）
func Init(cfg config.ZapLogConfig) error {
	var err error
	loggerInitOnce.Do(func() {
		// 解析日志级别（支持简写如dbg=debug、inf=info等）
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

		// 创建日志存储目录（权限0755：所有者读/写/执行，其他用户读/执行）
		if err = os.MkdirAll(cfg.Path, 0755); err != nil {
			return
		}

		// 初始化日志轮转器：按天轮转，保留7天日志
		writer, wErr := rotatelogs.New(
			filepath.Join(cfg.Path, "agent-%Y%m%d-000000.log"), // 日志文件名格式（按日期命名）
			rotatelogs.WithMaxAge(7*24*time.Hour),              // 日志最大保留时间（7天）
			rotatelogs.WithRotationTime(24*time.Hour),          // 轮转周期（24小时，即每天00:00轮转）
		)
		if wErr != nil {
			err = wErr
			return
		}

		// customTimeEncoderConsole 控制台输出的时间编码器：蓝色格式化时间（增强可读性）
		customTimeEncoderConsole := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(fmt.Sprintf("\033[34m%s\033[0m", t.Format("2006-01-02 15:04:05.000 -07:00")))
		}

		// customTimeEncoderJSON 文件存储的时间编码器：标准格式化时间（无颜色，适配JSON结构）
		customTimeEncoderJSON := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000 -07:00"))
		}

		// coloredLevelEncoder 控制台输出的级别编码器：不同级别日志显示不同颜色
		// debug(蓝)/info(绿)/warn(黄)/error(红)/panic/fatal(紫)，增强视觉区分度
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

		// 控制台编码器配置（开发环境友好，格式化输出）
		consoleEncoderCfg := zap.NewDevelopmentEncoderConfig()
		consoleEncoderCfg.ConsoleSeparator = " "                // 字段间分隔符（空格）
		consoleEncoderCfg.EncodeLevel = coloredLevelEncoder     // 启用彩色级别编码
		consoleEncoderCfg.EncodeTime = customTimeEncoderConsole // 启用彩色时间编码
		// 精简调用者信息：只保留"父目录/文件名:行号"（如logger/logger.go:123）
		consoleEncoderCfg.EncodeCaller = func(c zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			rel := filepath.Join(filepath.Base(filepath.Dir(c.File)), filepath.Base(c.File))
			enc.AppendString(fmt.Sprintf("%s:%d", rel, c.Line))
		}
		consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderCfg) // 控制台输出编码器

		// JSON编码器配置（生产环境友好，结构化存储）
		jsonCfg := zap.NewProductionEncoderConfig()
		jsonCfg.TimeKey = "timestamp"                       // JSON中时间字段的key（默认是ts，改为更直观的timestamp）
		jsonCfg.EncodeTime = customTimeEncoderJSON          // 时间格式统一
		jsonCfg.EncodeLevel = zapcore.LowercaseLevelEncoder // 级别字段小写（如debug/info）
		jsonEncoder := zapcore.NewJSONEncoder(jsonCfg)      // 文件存储编码器

		// 创建日志核心：同时输出到控制台和文件，按配置级别过滤
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level), // 控制台输出
			zapcore.NewCore(jsonEncoder, zapcore.AddSync(writer), level),       // 文件输出（轮转）
		)

		// 初始化基础日志实例：
		// - zap.AddCaller()：记录调用者信息（文件+行号）
		// - zap.AddCallerSkip(1)：跳过当前包装层，记录真实调用者
		// - zap.AddStacktrace(zapcore.ErrorLevel)：error及以上级别日志附加堆栈信息
		baseLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))
		loggerInitialized = true // 标记初始化完成
	})
	return err
}

// SetDefaultCollector 设置全局默认的collector字段值（线程安全）
// 适用于整个应用统一使用同一个采集器标识的场景，后续日志会自动携带该字段
// 参数collector：要设置的采集器名称（如"metrics-collector"、"log-collector"）
func SetDefaultCollector(collector string) {
	mu.Lock()
	defer mu.Unlock()
	defaultFields.Collector = collector
}

// GetDefaultCollector 获取当前全局默认的collector字段值（线程安全）
// 返回值：当前设置的默认采集器名称（未设置时为空字符串）
func GetDefaultCollector() string {
	mu.RLock()
	defer mu.RUnlock()
	return defaultFields.Collector
}

// getDefaultFields 内部辅助函数，获取日志默认字段切片
// 支持通过collectorOverride参数临时覆盖全局默认的collector字段
// 参数collectorOverride：可选参数，优先级高于全局默认collector（为空时使用全局默认）
// 返回值：包含collector和goroutine ID的默认字段切片
func getDefaultFields(collectorOverride ...string) []zapcore.Field {
	mu.RLock()
	collector := defaultFields.Collector
	mu.RUnlock()

	// 优先使用覆盖值（如果非空）
	if len(collectorOverride) > 0 && collectorOverride[0] != "" {
		collector = collectorOverride[0]
	}

	return []zapcore.Field{
		zap.String("collector", collector), // 采集器标识字段
		zap.String("goid", getGID()),       // 当前goroutine ID字段（用于并发调试）
	}
}

// getGID 内部辅助函数，获取当前goroutine的ID（字符串格式）
// 实现原理：通过runtime.Stack获取栈信息，解析goroutine ID
// 返回值：goroutine ID字符串（获取失败时返回"0"）
func getGID() string {
	var buf [64]byte
	n := runtime.Stack(buf[:], false) // 获取当前goroutine的栈信息（不包含其他goroutine）
	// 栈信息格式："goroutine 1 [running]: ..."，截取第一个空格后的数字即为ID
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))
	if len(idField) > 0 {
		if id, err := strconv.Atoi(idField[0]); err == nil {
			return strconv.Itoa(id)
		}
	}
	return "0" // 解析失败时返回默认值
}

// log 内部核心日志输出函数，统一处理各级别日志的输出逻辑
// 负责检查日志初始化状态、合并默认字段与自定义字段、按级别输出日志
// 参数level：日志级别（zap.DebugLevel/zap.InfoLevel等）
// 参数msg：日志消息内容（核心日志文本）
// 参数collectorOverride：临时覆盖当前日志的collector字段（为空时使用全局默认）
// 参数fields：额外的自定义日志字段（如zap.Int("code", 200)、zap.String("trace_id", "xxx")）
func log(level zapcore.Level, msg string, collectorOverride string, fields ...zapcore.Field) {
	if !loggerInitialized {
		panic("logger not initialized: call logger.Init() first") // 未初始化时panic，强制用户遵守使用规范
	}

	// 获取默认字段（支持覆盖collector）
	defaultFields := getDefaultFields(collectorOverride)

	// 构建最终日志实例：跳过当前log函数层、合并默认字段与自定义字段
	loggerWithFields := baseLogger.WithOptions(zap.AddCallerSkip(1)).With(defaultFields...).With(fields...)

	// 按级别输出日志（使用Check+Write模式，支持日志级别过滤优化）
	switch level {
	case zap.DebugLevel:
		if ce := loggerWithFields.Check(zap.DebugLevel, msg); ce != nil {
			ce.Write()
		}
	case zap.InfoLevel:
		if ce := loggerWithFields.Check(zap.InfoLevel, msg); ce != nil {
			ce.Write()
		}
	case zap.WarnLevel:
		if ce := loggerWithFields.Check(zap.WarnLevel, msg); ce != nil {
			ce.Write()
		}
	case zap.ErrorLevel:
		if ce := loggerWithFields.Check(zap.ErrorLevel, msg); ce != nil {
			ce.Write()
		}
	case zap.PanicLevel:
		if ce := loggerWithFields.Check(zap.PanicLevel, msg); ce != nil {
			ce.Write()
		}
	case zap.FatalLevel:
		if ce := loggerWithFields.Check(zap.FatalLevel, msg); ce != nil {
			ce.Write()
		}
	}
}

// Debug 输出Debug级别日志（最低级别，用于调试细节）
// 参数msg：日志消息内容
// 参数collectorOverride：可选，临时覆盖当前日志的collector字段（为空则使用全局默认）
// 参数fields：额外的自定义日志字段（支持zap提供的所有字段类型）
func Debug(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.DebugLevel, msg, collectorOverride, fields...)
}

// Info 输出Info级别日志（普通运行信息，如服务启动、关键流程执行）
// 参数msg：日志消息内容
// 参数collectorOverride：可选，临时覆盖当前日志的collector字段（为空则使用全局默认）
// 参数fields：额外的自定义日志字段（支持zap提供的所有字段类型）
func Info(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.InfoLevel, msg, collectorOverride, fields...)
}

// Warn 输出Warn级别日志（警告信息，如非致命错误、异常但不影响运行的场景）
// 参数msg：日志消息内容
// 参数collectorOverride：可选，临时覆盖当前日志的collector字段（为空则使用全局默认）
// 参数fields：额外的自定义日志字段（支持zap提供的所有字段类型）
func Warn(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.WarnLevel, msg, collectorOverride, fields...)
}

// Error 输出Error级别日志（错误信息，如业务异常、功能执行失败，会附加堆栈信息）
// 参数msg：日志消息内容
// 参数collectorOverride：可选，临时覆盖当前日志的collector字段（为空则使用全局默认）
// 参数fields：额外的自定义日志字段（支持zap提供的所有字段类型）
func Error(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.ErrorLevel, msg, collectorOverride, fields...)
}

// Panic 输出Panic级别日志（严重错误，输出后会触发panic）
// 参数msg：日志消息内容
// 参数collectorOverride：可选，临时覆盖当前日志的collector字段（为空则使用全局默认）
// 参数fields：额外的自定义日志字段（支持zap提供的所有字段类型）
func Panic(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.PanicLevel, msg, collectorOverride, fields...)
}

// Fatal 输出Fatal级别日志（致命错误，输出后会调用os.Exit(1)终止程序）
// 参数msg：日志消息内容
// 参数collectorOverride：可选，临时覆盖当前日志的collector字段（为空则使用全局默认）
// 参数fields：额外的自定义日志字段（支持zap提供的所有字段类型）
func Fatal(msg string, collectorOverride string, fields ...zapcore.Field) {
	log(zap.FatalLevel, msg, collectorOverride, fields...)
}

// Sync 同步日志缓冲区，确保所有待写入的日志数据都持久化到存储介质（文件/控制台）
// 建议在程序退出前调用（如main函数结束时），避免日志丢失
// 返回值：同步过程中产生的错误（如文件写入失败）
func Sync() error {
	if !loggerInitialized {
		return nil // 未初始化时直接返回nil，避免报错
	}
	return baseLogger.Sync()
}

// GetLogger 获取基础的zap.Logger实例（包含默认配置和字段）
// 适用于需要直接使用zap原生API的场景（如自定义日志输出逻辑）
// 注意：未初始化时调用会panic，需确保先执行Init()
// 返回值：初始化后的zap.Logger核心实例
func GetLogger() *zap.Logger {
	if !loggerInitialized {
		panic("logger not initialized: call logger.Init() first")
	}
	return baseLogger
}
