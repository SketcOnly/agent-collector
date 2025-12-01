package collector

import (
	"bufio"
	"fmt"
	"github.com/agent-collector/pkg/logger"
	"go.uber.org/zap"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LoadCalculator CPU负载计算器：不依赖gopsutil，原生实现1/5/15分钟负载滚动平均
type LoadCalculator struct {
	mu          sync.RWMutex  // 并发安全锁
	sampleCycle time.Duration // 采样周期（默认1秒）
	load1       float64       // 1 分钟负载平均值
	load5       float64       // 5 分钟负载平均值
	load15      float64       // 15 分钟负载平均值
	alpha1      float64       // 1分钟衰减系数（α=1-exp(-Δt/60)）
	alpha5      float64       // 5分钟衰减系数（α=1-exp(-Δt/300)）
	alpha15     float64       // 15分钟衰减系数（α=1-exp(-Δt/900)）
	initialized bool          // 初始化标记（首次采样后完成）
}

// NewLoadCalculator 创建负载计算器(采样周期建议1秒)
func NewLoadCalculator(sampleCycle time.Duration) (*LoadCalculator, error) {
	if sampleCycle <= 0 {
		return nil, fmt.Errorf("sample cycle must be positive (got %v)", sampleCycle)
	}
	dt := sampleCycle.Seconds()
	// 指数移动平均衰减系数，贴合linux内核逻辑
	return &LoadCalculator{
		sampleCycle: sampleCycle,
		alpha1:      1 - math.Exp(-dt/60),
		alpha5:      1 - math.Exp(-dt/300),
		alpha15:     1 - math.Exp(-dt/900),
	}, nil
}

// StartLoad Start 启动后台采集携程(非阻塞)
func (c *LoadCalculator) StartLoad() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("load collection only supports Linux (current OS: %s)", runtime.GOOS)
	}
	go func() {
		ticker := time.NewTicker(c.sampleCycle)
		defer ticker.Stop()

		for range ticker.C {
			if err := c.sampleAndUpdate(); err != nil {
				logger.Warn("load calculator sample failed", zap.Error(err))
			}
		}
	}()
	return nil
}

// GetLoads 获取当前1/5/15分钟负载（线程安全）
func (c *LoadCalculator) GetLoads() (load1, load5, load15 float64, initialized bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.load1, c.load5, c.load15, c.initialized
}

// sampleAndUpdate 采样瞬时负载并更新滚动平均
func (c *LoadCalculator) sampleAndUpdate() error {
	currentLoad, err := c.getCurrentLoad()
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.initialized {
		// 首次采样初始化负载值
		c.load1, c.load5, c.load15 = currentLoad, currentLoad, currentLoad
		c.initialized = true
		return nil
	}

	//	指数移动平均工时更新负载
	c.load1 = c.load1*(1-c.alpha1) + currentLoad*c.alpha1
	c.load5 = c.load5*(1-c.alpha5) + currentLoad*c.alpha5
	c.load15 = c.load15*(1-c.alpha15) + currentLoad*c.alpha15
	return nil
}

// getCurrentLoad 从/proc/loadavg读取瞬时负载
func (c *LoadCalculator) getCurrentLoad() (float64, error) {
	open, err := os.Open("/proc/loadavg")
	if err != nil {
		return 0, fmt.Errorf("open /proc/loadavg: %w", err)
	}
	defer open.Close()

	scanner := bufio.NewScanner(open)
	if !scanner.Scan() {
		return 0, fmt.Errorf("read /proc/loadavg: %w", scanner.Err())
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid /proc/loadavg format: %s", scanner.Err())
	}
	return strconv.ParseFloat(fields[0], 64)
}
