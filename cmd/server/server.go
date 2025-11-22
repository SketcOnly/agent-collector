package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/agent-collector/pkg/config"
	"github.com/agent-collector/pkg/logger"
	log "github.com/agent-collector/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Server HTTP服务实例，封装核心依赖和配置
type Server struct {
	cfg      *config.Config
	logger   *logger.Logger
	server   *http.Server
	registry *prometheus.Registry
	mux      *customMux
}

// statusWriter 包装ResponseWriter，捕获状态码
type statusWriter struct {
	http.ResponseWriter
	status int
}

// customMux 自定义Mux，兼容原生用法并记录路由
type customMux struct {
	http.ServeMux
	routes []string
	mu     sync.Mutex
}

const defaultShutdownTimeout = 5 * time.Second

// Handle 重写Handle，注册路由时记录路径
func (m *customMux) Handle(pattern string, handler http.Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, route := range m.routes {
		if route == pattern {
			m.ServeMux.Handle(pattern, handler)
			return
		}
	}
	
	m.routes = append(m.routes, pattern)
	m.ServeMux.Handle(pattern, handler)
}

// HandleFunc 重写HandleFunc
func (m *customMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.Handle(pattern, http.HandlerFunc(handler))
}

// NewHTTPServer 创建HTTP服务实例
func NewHTTPServer(cfg *config.Config, logger *logger.Logger, registry *prometheus.Registry) *Server {
	mux := &customMux{}
	
	srv := &Server{
		cfg:      cfg,
		logger:   logger,
		registry: registry,
		mux:      mux,
	}
	
	// 注册核心端点
	srv.registerEndpoints()
	
	srv.server = &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      srv.logMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	
	return srv
}

// logMiddleware 统一日志记录
func (s *Server) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		
		next.ServeHTTP(sw, r)
		
		s.logger.Info(
			"HTTP request",
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("remote", r.RemoteAddr),
			zap.Int("status", sw.status),
			zap.Duration("duration", time.Since(start)),
		)
	})
}

// registerEndpoints 注册核心路由
func (s *Server) registerEndpoints() {
	// 根路径 / 显示 HTML 页面，包含可点击的链接
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		
		html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html lang="zh-CN">
		<head>
			<meta charset="UTF-8">
			<title>Agent Collector</title>
			<style>
				body { font-family: Arial, sans-serif; margin: 40px; }
				h1 { color: #333; }
				a { display: block; margin: 8px 0; font-size: 18px; }
				code { background-color: #f0f0f0; padding: 2px 4px; }
			</style>
		</head>
		<body>
			<h1>Agent Collector Service</h1>
			<p>Version: <code>v1.0.0</code></p>
			<p>Service is running.</p>
			<h2>Available Endpoints:</h2>
			<a href="/health">/health - 健康检查</a>
			<a href="/metrics">/metrics - Prometheus 指标暴露</a>
		</body>
		</html>
		`)
		_, _ = w.Write([]byte(html))
	})
	
	// /metrics 端点
	s.mux.Handle("/metrics", promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{
		ErrorLog: zap.NewStdLog(log.GetGlobalLogger()),
	}))
	
	// /health 端点
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

// WriteHeader 捕获状态码
func (w *statusWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Start 启动HTTP服务（非阻塞）
func (s *Server) Start() error {
	s.logger.Info(
		"starting HTTP server",
		zap.String("listen_addr", s.cfg.Server.Addr),
		zap.Strings("handle_funcs", s.mux.routes),
	)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("HTTP server failed", zap.Error(err))
		}
	}()
	return nil
}

// Shutdown 优雅关闭HTTP服务
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()
	
	if err := s.server.Shutdown(ctx); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.logger.Warn("shutdown timeout exceeded")
			return nil
		}
		s.logger.Error("HTTP server shutdown failed", zap.Error(err))
		return err
	}
	
	s.logger.Info("HTTP server shutdown successfully")
	return nil
}

// WaitForShutdown 监听退出信号
func WaitForShutdown(shutdownFunc func() error) {
	if shutdownFunc == nil {
		log.Error("shutdownFunc is nil, cannot execute shutdown")
		return
	}
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	
	log.Info("service running, waiting for SIGINT/SIGTERM...")
	
	sig := <-sigChan
	log.Info("received shutdown signal", zap.String("signal", sig.String()))
	
	if err := shutdownFunc(); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	} else {
		log.Info("graceful shutdown completed successfully")
	}
	
	if err := log.Sync(); err != nil && err.Error() != "sync /dev/stdout: bad file descriptor" {
		log.Warn("logger sync failed", zap.Error(err))
	}
	
	log.Info("shutdown workflow finished, exiting program")
}
