// Package server 提供HTTP服务器核心功能，包含Prometheus指标暴露、健康检查端点、
// 优雅关闭机制及系统信号监听能力，用于支撑服务可观测性和高可用性。
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

// HTTPServer HTTP服务实例，封装监听地址、HTTP服务器核心对象和Prometheus指标注册器
// 核心能力：暴露/metrics指标端点、/health健康检查端点、优雅启动/关闭
type HTTPServer struct {
	addr     string               // 监听地址（格式：ip:port）
	server   *http.Server         // 底层HTTP服务器对象
	registry *prometheus.Registry // Prometheus指标注册器（注入自定义指标）
}

// statusWriter 包装http.ResponseWriter，用于捕获HTTP响应状态码
// 解决原生ResponseWriter无法直接获取返回状态码的问题
type statusWriter struct {
	http.ResponseWriter     // 嵌入原生ResponseWriter，继承其所有方法
	status              int // 记录响应状态码，默认200 OK
}

// httpShutdownTimeout 优雅关闭超时时间，避免关闭流程无限阻塞
const httpShutdownTimeout = 5 * time.Second

// NewHTTPServer 创建HTTP服务实例（依赖注入模式）
// 参数：
//
//	addr: 服务监听地址（例：":8080"）
//	registry: Prometheus指标注册器，用于暴露自定义指标
//
// 返回：
//
//	*HTTPServer: HTTP服务实例
//
// 核心初始化：
//  1. 注册/metrics端点：暴露Prometheus指标（含自定义指标）
//  2. 注册/health端点：提供服务健康检查（返回200 OK）
//  3. 配置HTTP超时参数（读/写/空闲超时）
func NewHTTPServer(addr string, registry *prometheus.Registry) *HTTPServer {
	mux := http.NewServeMux()
	
	// logRequest  请求日志记录辅助函数
	// 功能：记录请求方法、URL、客户端地址、响应状态码、处理耗时
	logRequest := func(r *http.Request, msg string, statusCode int, start time.Time) {
		logger.Info(
			msg,
			"",
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("remote", r.RemoteAddr),
			zap.Int("status", statusCode),
			zap.Duration("duration", time.Since(start)),
		)
	}
	
	// /metrics 端点：暴露Prometheus指标（含自定义注册器中的指标）
	mux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// 用statusWriter包装，捕获响应状态码
		ww := &statusWriter{ResponseWriter: w, status: 200}
		
		// 使用自定义Prometheus注册器创建指标处理器
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			ErrorLog: zap.NewStdLog(logger.GetLogger()), // 复用全局日志器
		}).ServeHTTP(ww, r)
		
		// 记录指标请求日志
		logRequest(r, "metrics request received", ww.status, start)
	}))
	
	// /health 端点：服务健康检查（无依赖检查，直接返回200 OK）
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: 200}
		
		ww.WriteHeader(http.StatusOK)
		_, _ = ww.Write([]byte("OK")) // 健康检查响应体
		
		// 记录健康检查日志
		logRequest(r, "health check received", ww.status, start)
	})
	
	return &HTTPServer{
		addr:     addr,
		registry: registry,
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,  // 读取请求体超时
			WriteTimeout: 10 * time.Second, // 写入响应超时
			IdleTimeout:  15 * time.Second, // 连接空闲超时
		},
	}
}

// WriteHeader 重写http.ResponseWriter的WriteHeader方法
// 功能：记录响应状态码到statusWriter实例中
func (w *statusWriter) WriteHeader(statusCode int) {
	w.status = statusCode                    // 保存状态码
	w.ResponseWriter.WriteHeader(statusCode) // 调用原生方法写入状态码
}

// Start 启动HTTP服务（非阻塞模式）
// 功能：在子goroutine中启动服务监听，不阻塞主流程
// 返回：
//
//	error: 启动阶段错误（实际监听错误在子goroutine中记录）
//
// 注意：
//  1. 服务启动后会持续运行，直到调用Shutdown方法
//  2. 非正常关闭（非http.ErrServerClosed）会触发Fatal日志
func (s *HTTPServer) Start() error {
	logger.Info(
		"starting HTTP server",
		"",
		zap.String("listen_addr", s.addr),
		zap.Duration("read_timeout", s.server.ReadTimeout),
		zap.Duration("write_timeout", s.server.WriteTimeout),
		zap.Duration("idle_timeout", s.server.IdleTimeout),
	)
	
	// 子goroutine中启动服务（避免阻塞主流程）
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			// 区分正常关闭和异常错误
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

// Shutdown 优雅关闭HTTP服务
// 功能：
//  1. 停止接收新请求
//  2. 等待现有请求在超时时间内处理完成
//  3. 释放服务资源
//
// 返回：
//
//	error: 关闭失败错误（超时错误会被忽略）
//
// 注意：
//   - 必须配合信号监听调用（如WaitForShutdown）
//   - 超时时间固定为httpShutdownTimeout（5秒）
func (s *HTTPServer) Shutdown() error {
	logger.Info("starting graceful shutdown of HTTP server", "", zap.String("listen_addr", s.addr))
	
	// 创建带超时的关闭上下文
	shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
	defer cancel() // 确保上下文资源释放
	
	// 执行优雅关闭
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		// 忽略超时错误（超时视为关闭完成）
		if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
			return nil
		}
		logger.Error("HTTP server shutdown failed", "", zap.Error(err), zap.String("listen_addr", s.addr))
		return err
	}
	logger.Info("HTTP server shutdown successfully", "", zap.String("listen_addr", s.addr))
	return nil
}

// WaitForShutdown 监听系统退出信号，触发优雅关闭流程
// 功能：
//  1. 监听SIGINT（Ctrl+C）和SIGTERM（容器停止信号）
//  2. 收到信号后执行自定义关闭逻辑
//  3. 处理关闭超时和日志同步
//
// 参数：
//
//	collector: 日志收集器名称（用于日志标识）
//	shutdownFunc: 自定义关闭函数（如HTTP服务关闭、资源清理等）
//
// 注意：
//   - shutdownFunc不能为nil，否则会记录错误并返回
//   - 关闭超时时间为5秒，超时后强制退出
func WaitForShutdown(collector string, shutdownFunc func() error) {
	// 容错：检查关闭函数是否为空
	if shutdownFunc == nil {
		logger.Error("shutdownFunc is nil, cannot execute shutdown", "")
		return
	}
	
	// 注册信号监听通道（缓冲大小1，避免信号丢失）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM) // 仅监听指定信号
	defer signal.Stop(sigChan)                              // 确保程序退出前解除信号监听
	
	// 日志提示：服务正常运行中
	logger.Info("service is running, waiting for shutdown signal (SIGINT/SIGTERM)...", "")
	
	// 阻塞等待信号（程序核心运行阶段）
	sig := <-sigChan
	logger.Info("received shutdown signal", "", zap.String("signal", sig.String()))
	
	// 执行关闭逻辑（带超时控制）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 异步执行关闭函数（避免阻塞信号处理）
	shutdownErrChan := make(chan error, 1)
	go func() {
		logger.Info("starting graceful shutdown...", "")
		shutdownErrChan <- shutdownFunc() // 执行自定义关闭逻辑（如HTTP服务Shutdown）
		close(shutdownErrChan)
	}()
	
	// 等待关闭完成或超时
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
	
	// 日志同步：确保缓存日志写入输出（忽略stdout无效句柄错误）
	if err := logger.Sync(); err != nil {
		if err.Error() != "sync /dev/stdout: bad file descriptor" {
			logger.Warn("logger sync failed", "", zap.Error(err))
		}
	}
	
	logger.Info("shutdown workflow finished, program exiting", "")
}
