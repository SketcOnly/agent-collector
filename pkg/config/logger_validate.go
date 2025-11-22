package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//Validate 规则说明
//字段	已通过 tag 校验	额外业务校验
//Level	oneof 预校验	再进行 map lookup，避免大小写或隐藏错误
//Format	oneof=json console	无
//Path	required	可写目录，自动创建
//MaxSize	gt=0	无
//MaxBackup	gte=0	无
//MaxAge	gte=0	无
//Compress	bool	无

// Validate 日志配置校验
func (l *ZapLogConfig) Validate() error {

	// --- 基础 tag 校验 ---
	if err := valid.Struct(l); err != nil {
		return fmt.Errorf("日志配置字段非法: %w", err)
	}

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
	abs, err := filepath.Abs(l.Path)
	if err != nil {
		return fmt.Errorf("Log.Path Failed to parse the log path (expected: :path), got %s: %w", l.Path, err)
	}
	if err := ensureDir(abs); err != nil {
		return fmt.Errorf("Log.Path The log directory is not writable (expected: :path), got %s: %w", l.Path, err)
	}
	return nil
}

func ensureDir(path string) error {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil

}
