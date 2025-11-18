package server

import (
	"context"
	"errors"
	"github.com/agent-collector/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HTTPServer HTTP服务（暴露指标+健康检查）
type HTTPServer struct {
	addr        string
	server      *http.Server
	collector   string
	goroutineid string
	registry    *prometheus.Registry
}

// 超时时间
const httpShutdownTimeout = 5 * time.Second

// NewHTTPServer 创建HTTP服务（注入自定义Prometheus注册器）
func NewHTTPServer(addr, collector string, registry *prometheus.Registry) *HTTPServer {

	// 4, 设置当前goid
	//logger.SetDefaultGid(strconv.FormatUint(goid.GetGID(), 10))
	//goroutineid := logger.GetDefaultGid()

	mux := http.NewServeMux()
	// 设置collector
	// 暴露Prometheus指标（使用自定义注册器）
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog: zap.NewStdLog(logger.GetLogger()),
	}))

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		logger.Info(
			"health check received",
			"",
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("remote", r.RemoteAddr),
		)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			return
		}
	})

	return &HTTPServer{
		addr:      addr,
		collector: collector,
		registry:  registry,
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}
}

// Start 启动HTTP服务（非阻塞）
func (s *HTTPServer) Start() error {

	logger.Info(
		"starting HTTP server",
		"",
		zap.String("listen_addr", s.addr),
		zap.Duration("read_timeout", s.server.ReadTimeout),
		zap.Duration("write_timeout", s.server.WriteTimeout),
		zap.Duration("idle_timeout", s.server.IdleTimeout),
	)

	//启动http 服务器(子 goroutine中运行，不阻塞主流程)
	go func() {
		//currentGID := strconv.FormatUint(goid.GetGID(), 10) // 子goroutine自己的GID
		if err := s.server.ListenAndServe(); err != nil {
			// 仅在非正常关闭时记录错误(ErrServerClosed是优雅关闭的正常错误)
			if !errors.Is(http.ErrServerClosed, err) {
				logger.Fatal(

					"HTTP server failed to listen",
					"",
					zap.Error(err),
					zap.String("listen_addr", s.addr),
				)
			} else {
				logger.Info(

					"HTTP server stopped listening",
					"",
					zap.String("listen_addr", s.addr),
				)
			}
		}
	}()
	return nil
}

// Shutdown 优雅关闭HTTP服务（必须实现，配合主程序信号监听）
func (s *HTTPServer) Shutdown() error {
	//logger.Info(s.collector, s.goroutineid, "starting graceful shutdown of HTTP server", zap.String("listen_addr", s.addr))

	// 创建关闭上下文（5秒超时，避免无限等待）
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭HTTP服务（停止接收新请求，等待现有请求处理完成）
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
			return nil
		}
		logger.Error("HTTP server shutdown failed", "", zap.Error(err), zap.String("listen_addr", s.addr))
		return err
	}
	logger.Info("HTTP server shutdown successfully", "", zap.String("listen_addr", s.addr))
	return nil
}

// WaitForShutdown 监听退出信号（SIGINT/SIGTERM），执行优雅关闭
// 参数：shutdownFunc - 自定义关闭逻辑（如服务器关闭、资源清理等）

func WaitForShutdown(collector string, shutdownFunc func() error) {
	// 步骤1：校验shutdownFunc非空（容错）
	if shutdownFunc == nil {
		//collector := logger.GetDefaultCollector()
		//gidStr := logger.GetDefaultGid()
		logger.Error("shutdownFunc is nil, cannot execute shutdown", "")
		return
	}

	// 步骤2：注册信号监听（先准备好接收信号）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan) // 确保后续解除监听

	// 步骤3：打印“等待信号”日志（此时程序开始阻塞，服务正常运行）
	logger.Info("service is running, waiting for shutdown signal (SIGINT/SIGTERM)...", "")

	// 步骤4：阻塞等待信号（关键！程序会停在这里，直到收到Ctrl+C/SIGTERM）
	sig := <-sigChan
	logger.Info("received shutdown signal", "", zap.String("signal", sig.String()))

	// 步骤5：收到信号后，才执行关闭逻辑（终于到这步！）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErrChan := make(chan error, 1)
	go func() {
		// 4, 设置当前goid
		//currentGID := strconv.FormatUint(goid.GetGID(), 10) // 子goroutine自己的GID
		logger.Info("starting graceful shutdown...", "")
		shutdownErrChan <- shutdownFunc() // 执行HTTP服务关闭
		close(shutdownErrChan)
	}()

	// 步骤6：等待关闭完成或超时
	select {
	case err := <-shutdownErrChan:
		if err != nil {
			logger.Error("graceful shutdown failed", "", zap.Error(err))
		} else {
			logger.Info("graceful shutdown completed successfully", "")
		}
	case <-ctx.Done():
		logger.Error("graceful shutdown timed out", "", zap.Error(ctx.Err()))
	}

	// 步骤7：日志同步（次要：处理stdout同步失败问题）

	if err := logger.Sync(); err != nil {
		// 忽略stdout的同步失败（zap对stdout的Sync可能无效）
		if err.Error() != "sync /dev/stdout: bad file descriptor" {
			logger.Warn("logger sync failed", "", zap.Error(err))
		}
	}

	logger.Info("shutdown workflow finished, program exiting", "")
}
