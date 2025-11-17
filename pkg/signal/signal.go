package signal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"go.uber.org/zap"
)

// WaitForShutdown 监听退出信号（SIGINT/SIGTERM），执行优雅关闭
func WaitForShutdown(logger *zap.Logger, shutdownFunc func() error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// 阻塞等待信号
	sig := <-sigChan
	logger.Info("received shutdown signal", zap.String("signal", sig.String()))
	
	// 超时控制关闭逻辑
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	go func() {
		if err := shutdownFunc(); err != nil {
			logger.Error("shutdown failed", zap.Error(err))
		}
		cancel()
	}()
	
	<-ctx.Done()
	logger.Info("shutdown completed")
}
