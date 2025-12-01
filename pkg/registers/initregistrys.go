package registers

import (
	"context"
	"fmt"
	"github.com/agent-collector/pkg/logger"
	"go.uber.org/zap"
	"sync"
	"time"
)

// AgentImpl 实现 registers.Agent 接口
type AgentImpl struct {
	collectors []Collector
	interval   time.Duration
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
}

//// GetRegisteredCollectors 返回所有已注册的采集器（返回副本，避免外部修改）
//func (r *Registry) GetRegisteredCollectors() []Collector {
//	copied := make([]Collector, len(r.collectors))
//	copy(copied, r.collectors)
//	return copied
//}

// NewRegistry 创建采集器注册器（初始化上下文）
func NewRegistry(interval time.Duration) *AgentImpl {
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentImpl{
		collectors: make([]Collector, 0),
		interval:   interval,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Register 注册采集器
func (r *AgentImpl) Register(c Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors = append(r.collectors, c)
}

// InitAll 辅助方法（修复逻辑+优化）
func (r *AgentImpl) InitAll() error {
	for _, coll := range r.collectors { // 重命名循环变量，避免覆盖
		if err := coll.Init(); err != nil {
			return fmt.Errorf("collector init failed: %w", err)
		}
		logger.Debug("collector initialized successfully", zap.String("name", coll.Name()))
	}
	return nil
}

// Start 启动采集器（移除冗余参数，关联外部上下文）
func (r *AgentImpl) Start(ctx context.Context) {
	// 初始化所有已注册采集器（初始化失败直接终止）
	if err := r.InitAll(); err != nil {
		logger.Fatal("failed to init all collectors", zap.String("name", "collector-registry"), zap.Error(err))
	}

	// 启动定时器
	r.ticker = time.NewTicker(r.interval)
	logger.Debug("collector metrics started", zap.String("name", "collector-registry"),
		zap.Duration("interval", r.interval),
		zap.Int("registered-collectors-count", len(r.collectors)))

	// 异步采集：同时监听外部 ctx 和内部 ctx 的关闭信号
	go func() {
		// 首次采集（失败仅警告）
		if err := r.CollectAll(ctx); err != nil {
			logger.Warn("first collection failed", zap.String("name", "collector-registry"), zap.Error(err))
		}

		for {
			select {
			case <-r.ticker.C:
				_ = r.CollectAll(ctx) // 单采集器失败不影响整体
			case <-ctx.Done(): // 响应外部关闭信号（如服务停止）
				r.ticker.Stop()
				logger.Info("collector metrics stopped by external context", zap.String("name", "collector-registry"), zap.Error(ctx.Err()))
				return
			case <-r.ctx.Done(): // 响应内部关闭信号（如主动调用 Shutdown）
				r.ticker.Stop()
				logger.Info("collector metrics stopped by internal shutdown", zap.String("name", "collector-registry"))
				return
			}
		}
	}()
}

// Shutdown 优雅关闭采集器（释放资源）
func (r *AgentImpl) Shutdown(ctx context.Context) error {
	logger.Info("starting to shutdown collector metrics", zap.String("name", "collector-registry"))

	// 停止定时器（双重保险）
	if r.ticker != nil {
		r.ticker.Stop()
	}

	// 触发内部上下文取消，终止采集循环
	r.cancel()

	// 关闭所有采集器，返回最后一个错误
	return r.CloseAll()
}

// CollectAll 批量采集数据（优化日志输出）
func (r *AgentImpl) CollectAll(ctx context.Context) error {
	var hasErr bool
	for _, collector := range r.collectors {
		if err := collector.Collect(ctx); err != nil {
			logger.Warn("collection failed", zap.String("name", collector.Name()), zap.Error(err))
			hasErr = true
		}
	}
	if hasErr {
		return fmt.Errorf("some collectors failed to collect data")
	}
	return nil
}

// CloseAll 批量关闭采集器（优化错误收集）
func (r *AgentImpl) CloseAll() error {
	var lastErr error
	for _, collector := range r.collectors {
		logger.Debug("closing collector", zap.String("name", collector.Name()))
		if err := collector.Close(); err != nil {
			logger.Error("failed to close collector", zap.String("name", collector.Name()), zap.Error(err))
			lastErr = err // 记录最后一个错误，不阻断整体关闭
		} else {
			logger.Debug("collector closed successfully", zap.String("name", collector.Name()))
		}
	}
	return lastErr
}
