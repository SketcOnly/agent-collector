package collector

import (
	"context"
)

// Agent 顶层采集器接口（封装所有采集器的生命周期管理）
// 后续扩展采集器仅需实现Collector接口，通过Agent注册即可
type Agent interface {
	Register(collector Collector)       // 注册采集器
	Start(ctx context.Context)          // 启动采集（定时器循环）
	Shutdown(ctx context.Context) error // 优雅停止
}
