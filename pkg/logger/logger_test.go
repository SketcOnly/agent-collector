package logger_test

import (
	"testing"
	"time"

	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// mockFatalHook 捕获 fatal 日志（不退出进程）
type mockFatalHook struct {
	called bool
}

func (h *mockFatalHook) Hook(e zapcore.Entry) error {
	if e.Level == zapcore.FatalLevel {
		h.called = true
	}
	return nil
}

func TestLoggerLevels(t *testing.T) {
	cfg := &config.ZapLogConfig{
		Level: "debug",
		Path:  "./test-logs",
	}

	_, err := logger.InitLogger(cfg)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	// 普通日志
	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	// Panic 测试
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic, but no panic occurred")
			}
		}()
		logger.Panic("panic msg")
	}()

	// Fatal 测试（使用 zap.Hooks，不触发 os.Exit）
	hook := &mockFatalHook{}
	l := logger.GetGlobalLogger().WithOptions(zap.Hooks(hook.Hook))
	l.Fatal("fatal msg")

	if !hook.called {
		t.Errorf("fatal hook was not triggered")
	}

	if err := logger.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
}
