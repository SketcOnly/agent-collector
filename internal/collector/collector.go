package collector

import (
	"context"
	"fmt"

	"github.com/agent-collector/logger"
	"go.uber.org/zap"
	"time"
)

// Collector 采集器核心接口（所有采集器必须实现）
type Collector interface {
	Name() string                      // 采集器名称（唯一标识）
	Init() error                       // 初始化（注册指标、预检查资源）
	Collect(ctx context.Context) error // 采集数据（更新指标）
	Close() error                      // 关闭（释放资源）
}

// Registry 采集器注册器（严格实现 Agent 接口）
type Registry struct {
	collectors []Collector
	interval   time.Duration
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
}

// GetRegisteredCollectors 返回所有已注册的采集器（返回副本，避免外部修改）
func (r *Registry) GetRegisteredCollectors() []Collector {
	copied := make([]Collector, len(r.collectors))
	copy(copied, r.collectors)
	return copied
}

// NewRegistry 创建采集器注册器（初始化上下文）
func NewRegistry(interval time.Duration) *Registry {
	ctx, cancel := context.WithCancel(context.Background())
	return &Registry{
		collectors: make([]Collector, 0),
		interval:   interval,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Register 严格实现 Agent 接口
// Register 注册采集器（补充名称唯一校验，避免重复）
func (r *Registry) Register(c Collector) {
	// 校验采集器名称唯一
	for _, existing := range r.collectors {
		if existing.Name() == c.Name() {
			logger.Warn("collector already registered, skip", c.Name(), zap.String("collector", c.Name()))
			return
		}
	}

	r.collectors = append(r.collectors, c)
	logger.Info("registered collector successfully", c.Name(), zap.String("collector", c.Name()))
}

// Start 启动采集器（移除冗余参数，关联外部上下文）
func (r *Registry) Start(ctx context.Context) {
	// 初始化所有已注册采集器（初始化失败直接终止）
	if err := r.InitAll(); err != nil {
		logger.Fatal("failed to init all collectors", "collector-registry", zap.Error(err))
	}

	// 启动定时器
	r.ticker = time.NewTicker(r.interval)
	logger.Info("collector agent started", "collector-registry",
		zap.Duration("interval", r.interval),
		zap.Int("registered-collectors-count", len(r.collectors)))

	// 异步采集：同时监听外部 ctx 和内部 ctx 的关闭信号
	go func() {
		// 首次采集（失败仅警告）
		if err := r.CollectAll(ctx); err != nil {
			logger.Warn("first collection failed", "collector-registry", zap.Error(err))
		}

		for {
			select {
			case <-r.ticker.C:
				_ = r.CollectAll(ctx) // 单采集器失败不影响整体
			case <-ctx.Done(): // 响应外部关闭信号（如服务停止）
				r.ticker.Stop()
				logger.Info("collector agent stopped by external context", "collector-registry", zap.Error(ctx.Err()))
				return
			case <-r.ctx.Done(): // 响应内部关闭信号（如主动调用 Shutdown）
				r.ticker.Stop()
				logger.Info("collector agent stopped by internal shutdown", "collector-registry")
				return
			}
		}
	}()
}

// Shutdown 优雅关闭采集器（释放资源）
func (r *Registry) Shutdown(ctx context.Context) error {
	logger.Info("starting to shutdown collector agent", "collector-registry")

	// 停止定时器（双重保险）
	if r.ticker != nil {
		r.ticker.Stop()
	}

	// 触发内部上下文取消，终止采集循环
	r.cancel()

	// 关闭所有采集器，返回最后一个错误
	return r.CloseAll()
}

// InitAll 辅助方法（修复逻辑+优化）
// InitAll 批量初始化所有采集器（移除冗余参数，修复变量覆盖）
func (r *Registry) InitAll() error {
	for _, collector := range r.collectors { // 重命名循环变量，避免覆盖
		logger.Debug("initializing collector", collector.Name())
		if err := collector.Init(); err != nil {
			logger.Error("failed to init collector", collector.Name(), zap.Error(err))
			return fmt.Errorf("collector %s init failed: %w", collector.Name(), err)
		}
		logger.Info("collector initialized successfully", collector.Name())
	}
	return nil
}

// CollectAll 批量采集数据（优化日志输出）
func (r *Registry) CollectAll(ctx context.Context) error {
	var hasErr bool
	for _, collector := range r.collectors {
		if err := collector.Collect(ctx); err != nil {
			logger.Warn("collection failed", collector.Name(), zap.Error(err))
			hasErr = true
		}
	}
	if hasErr {
		return fmt.Errorf("some collectors failed to collect data")
	}
	return nil
}

// CloseAll 批量关闭采集器（优化错误收集）
func (r *Registry) CloseAll() error {
	var lastErr error
	for _, collector := range r.collectors {
		logger.Debug("closing collector", collector.Name())
		if err := collector.Close(); err != nil {
			logger.Error("failed to close collector", collector.Name(), zap.Error(err))
			lastErr = err // 记录最后一个错误，不阻断整体关闭
		} else {
			logger.Info("collector closed successfully", collector.Name())
		}
	}
	return lastErr
}
