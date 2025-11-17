package config

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	HTTP      HTTPServerConfig `yaml:"http"`
	Collector CollectorConfig  `yaml:"collector"`
	Log       ZapLogConfig     `yaml:"log"`
}

// HTTPServerConfig HTTP服务配置
type HTTPServerConfig struct {
	Addr string `yaml:"addr" mapstructure:"addr" env:"HTTP_ADDR"`
}

// CPUCollectorConfig CPU采集器配置
type CPUCollectorConfig struct {
	CPUCollectorConfig bool `yaml:"collect_per_core" mapstructure:"collect_per_core" env:"CPU_COLLECT_PER_CORE"`
}

// CollectorConfig 采集器全局配置
type CollectorConfig struct {
	Interval       time.Duration      `yaml:"interval" mapstructure:"interval" env:"COLLECT_INTERVAL"`
	EnableDisk     bool               `yaml:"enable_disk" mapstructure:"enable_disk" env:"ENABLE_DISK"`
	EnableNetwork  bool               `yaml:"enable_network" mapstructure:"enable_network" env:"ENABLE_NETWORK"`
	EnableCPU      bool               `yaml:"enable_cpu" mapstructure:"enable_cpu" env:"ENABLE_CPU"`
	IgnoreDisks    []string           `yaml:"ignore_disks" mapstructure:"ignore_disks"`
	IgnoreNetworks []string           `yaml:"ignore_networks" mapstructure:"ignore_networks"`
	CPU            CPUCollectorConfig `yaml:"cpu" mapstructure:"cpu"`
}

// ZapLogConfig 全局日志配置
type ZapLogConfig struct {
	Level  string `yaml:"level" mapstructure:"level" env:"LOG_LEVEL"`
	Format string `yaml:"format" mapstructure:"format"`
	Path   string `yaml:"path" mapstructure:"path"`
}

// Load 加载配置 （优先级：命令行指定文件 > 默认config.yaml > 环境变量）
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 	基础配置
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.AllowEmptyEnv(true)

	// 	配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigFile("config.yaml")
		v.AddConfigPath(".")
	}

	// 	读取配置
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("[ERROR] read config failed: %w", err)
	}
	// 	设置默认值
	setDefaults(v)
	// 	解析到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("[ERROR] unmarshal config failed: %w", err)
	}

	// 	配置校验
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("[ERROR] config validate failed: %w", err)
	}
	return &cfg, nil

}

// Validate 配置校验
func (c *Config) Validate() error {
	// 	1,校验HTTP服务配置
	if err := c.HTTP.Validate(); err != nil {
		return err
	}

	// 	2，校验采集配置
	if err := c.Collector.Validate(); err != nil {
		return err
	}
	// 	3，校验日志配置
	if err := c.Log.Validate(); err != nil {
		return err
	}
	return nil
}

// Validate HTTP服务配置校验
func (h *HTTPServerConfig) Validate() error {
	// 	校验Addr格式(必须是 ":port" 或 "ip:port")
	if h.Addr == "" {
		return errors.New("[ERROR] HTTP.Addr cannot be empty")
	}
	// 	用net包解析地址，验证格式合法性
	_, err := net.ResolveTCPAddr("tcp", h.Addr)
	if err != nil {
		return fmt.Errorf("[ERROR] HTTP.Addr format invalid (expected: :port or ip:port), got %s: %w", h.Addr, err)
	}
	return nil
}

// Validate 采集器配置校验

func (col *CollectorConfig) Validate() error {
	// 	校验采集间隔，(最小1秒，最大1小时，避免过频/过久)
	if col.Interval < time.Second || col.Interval > 3600*time.Second {
		return fmt.Errorf("[ERROR] Collector.Interval must be between 1s and 3600s, got %v", col.Interval)
	}
	// 	校验至少启用一个采集器，否则没有意义
	if !col.EnableDisk && !col.EnableCPU && !col.EnableNetwork {
		return fmt.Errorf("[ERROR] at loast one collector must be enabled (EnableDisk/EnableNetwork/EnableCPU)")
	}
	// 	校验忽略列表(不能包含空字符串)
	for _, disk := range col.IgnoreDisks {
		if strings.TrimSpace(disk) == "" {
			return errors.New("[ERROR] Collector.IgnoreDisks cannot contain empty string")
		}
	}
	for _, iface := range col.IgnoreNetworks {
		if strings.TrimSpace(iface) == "" {
			return errors.New("[ERROR] Collector.IgnoreNetworks cannot contain empty string")
		}
	}
	return nil
}

// Validate 日志配置校验
func (l *ZapLogConfig) Validate() error {
	// 	校验日志级别，（必须是zap支持的合法级别）
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[strings.ToLower(l.Level)] {
		return fmt.Errorf("Log.Level invalid (valid: debug/info/warn/error), got %s", l.Level)
	}
	// 	校验日志格式
	if l.Format != "json" && l.Format != "console" {
		return fmt.Errorf("Log.Format must be 'json' or 'console', got %s", l.Format)
	}
	// 	校验日志路径(非空，确保可创建)
	if strings.TrimSpace(l.Path) == "" {
		return errors.New("Log.Path cannot be empty")
	}
	return nil
}

// setDefaults 配置默认值
func setDefaults(v *viper.Viper) {
	v.SetDefault("http.addr", ":9091")
	v.SetDefault("collector.interval", "10s")
	v.SetDefault("collector.enable_disk", true)
	v.SetDefault("collector.enable_network", true)
	v.SetDefault("collector.enable_cpu", true)
	v.SetDefault("collector.ignore_disks", []string{"loop", "ram", "tmpfs"})
	v.SetDefault("collector.ignore_networks", []string{"lo"})
	v.SetDefault("collector.cpu.collect_per_core", true)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.path", "./logs/agent.log")
}
