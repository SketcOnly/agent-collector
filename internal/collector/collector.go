package collector

import (
	"context"
	"github.com/agent-collector/pkg/logger"
	"time"

	"go.uber.org/zap"
)

// Collector 采集器核心接口（所有采集器必须实现）
type Collector interface {
	Name() string                      // 采集器名称（唯一标识）
	Init() error                       // 初始化（注册指标、预检查资源）
	Collect(ctx context.Context) error // 采集数据（更新指标）
	Close() error                      // 关闭（释放资源）
}

// Registry 采集器注册器（实现Agent接口）
type Registry struct {
	collectors       []Collector
	defaultCollector string
	interval         time.Duration
	ticker           *time.Ticker
	ctx              context.Context
	cancel           context.CancelFunc
}

// NewRegistry 创建采集器注册器
func NewRegistry(interval time.Duration) *Registry {
	ctx, cancel := context.WithCancel(context.Background())
	return &Registry{
		collectors: make([]Collector, 0),
		interval:   interval,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// -------------------------- 实现Agent接口 --------------------------
func (r *Registry) Register(c Collector) {
	r.collectors = append(r.collectors, c)
	logger.Info("registered collector", "", zap.String("collector", c.Name()))
}

func (r *Registry) Start(ctx context.Context) {
	// 初始化所有采集器
	if err := r.InitAll(); err != nil {
		logger.Fatal("failed to init collectors", "", zap.Error(err))
	}

	// 启动定时器
	r.ticker = time.NewTicker(r.interval)
	logger.Info("collector agent started", "", zap.Duration("interval", r.interval))

	// 循环采集（首次启动立即采集）
	go func() {
		if err := r.CollectAll(ctx); err != nil {
			logger.Warn("first collect failed", "", zap.Error(err))
		}

		for {
			select {
			case <-r.ticker.C:
				_ = r.CollectAll(ctx) // 单采集器失败不影响整体
			case <-r.ctx.Done():
				r.ticker.Stop()
				logger.Info("collector agent stopped", "")
				return
			}
		}
	}()
}

func (r *Registry) Shutdown(ctx context.Context) error {
	logger.Info("shutting down collector agent", "")
	r.cancel()
	return r.CloseAll()
}

// -------------------------- 辅助方法 --------------------------
func (r *Registry) InitAll() error {
	for _, c := range r.collectors {
		if err := c.Init(); err != nil {
			logger.Error("failed to init collector", "", zap.String("collector", c.Name()), zap.Error(err))
			return err
		}
		logger.Info("initialized collector", "", zap.String("collector", c.Name()))
	}
	return nil
}

func (r *Registry) CollectAll(ctx context.Context) error {
	for _, c := range r.collectors {
		if err := c.Collect(ctx); err != nil {
			logger.Warn("failed to collect data", "", zap.String("collector", c.Name()), zap.Error(err))
			continue
		}
	}
	return nil
}

func (r *Registry) CloseAll() error {
	var lastErr error
	for _, c := range r.collectors {
		if err := c.Close(); err != nil {
			logger.Error("failed to close collector", "", zap.String("collector", c.Name()), zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}
