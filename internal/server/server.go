package server

import (
	"context"
	"fmt"
	"github.com/agent-collector/pkg/goid"
	"github.com/agent-collector/pkg/logger"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HTTPServer HTTP服务（暴露指标+健康检查）
type HTTPServer struct {
	addr             string
	defaultCollector string
	server           *http.Server
	registry         *prometheus.Registry
}

// NewHTTPServer 创建HTTP服务（注入自定义Prometheus注册器）
func NewHTTPServer(addr string, registry *prometheus.Registry) *HTTPServer {
	mux := http.NewServeMux()
	// 设置collector
	// 暴露Prometheus指标（使用自定义注册器）
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog: zap.NewStdLog(logger.GetLogger()),
	}))

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("http-collector", fmt.Sprintf("%d", goid.GetGID()),
			"health check received",
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("remote", r.RemoteAddr),
		)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return &HTTPServer{
		addr:             addr,
		defaultCollector: "http-collector",
		registry:         registry,
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
	logger.Info(s.defaultCollector, fmt.Sprintf("%d", goid.GetGID()),
		"starting HTTP server",
		zap.String("listen_addr", s.addr),
		zap.Duration("read_timeout", s.server.ReadTimeout),
		zap.Duration("write_timeout", s.server.WriteTimeout),
		zap.Duration("idle_timeout", s.server.IdleTimeout),
	)

	//启动http 服务器(子 goroutine中运行，不阻塞主流程)
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			// 仅在非正常关闭时记录错误(ErrServerClosed是优雅关闭的正常错误)
			if err != http.ErrServerClosed {
				logger.Fatal(
					s.defaultCollector,
					fmt.Sprintf("%d", goid.GetGID()),
					"HTTP server failed to listen",
					zap.Error(err),
					zap.String("listen_addr", s.addr),
				)
			} else {
				logger.Info(
					s.defaultCollector,
					fmt.Sprintf("%d", goid.GetGID()),
					"HTTP server stopped listening",
					zap.String("listen_addr", s.addr),
				)
			}
		}
	}()
	return nil
}

// Shutdown 优雅关闭HTTP服务（必须实现，配合主程序信号监听）
func (s *HTTPServer) Shutdown() error {
	logger.Info(
		s.defaultCollector,
		fmt.Sprintf("%d", goid.GetGID()),
		"starting graceful shutdown of HTTP server",
		zap.String("listen_addr", s.addr),
	)

	// 创建关闭上下文（5秒超时，避免无限等待）
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭HTTP服务（停止接收新请求，等待现有请求处理完成）
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		logger.Error(
			s.defaultCollector,
			fmt.Sprintf("%d", goid.GetGID()),
			"HTTP server shutdown failed",
			zap.Error(err),
			zap.String("listen_addr", s.addr),
		)
		return err
	}

	logger.Info(
		s.defaultCollector,
		fmt.Sprintf("%d", goid.GetGID()),
		"HTTP server shutdown successfully",
		zap.String("listen_addr", s.addr),
	)
	return nil
}
