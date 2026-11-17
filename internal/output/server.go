// Package output 负责采集结果的落地输出：控制台、日志、HTTP JSON API。
package output

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"MPulse/internal/logger"
	"MPulse/internal/scheduler"
)

// Server 基于 gin 的轻量 HTTP 服务，仅返回 JSON。
type Server struct {
	sched  *scheduler.Scheduler
	engine *gin.Engine
	srv    *http.Server
	start  time.Time
}

// NewServer 构造 HTTP 服务，addr 形如 :55055。
func NewServer(addr string, sched *scheduler.Scheduler) *Server {
	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		sched:  sched,
		start:  time.Now(),
		engine: gin.New(),
	}

	s.engine.Use(requestLogger(), gin.Recovery())
	s.registerRoutes()

	s.srv = &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// registerRoutes 注册全部路由。
func (s *Server) registerRoutes() {
	r := s.engine

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    "MPulse",
			"version": "1.0.0",
			"endpoints": []string{
				"GET /api/v1/health",
				"GET /api/v1/metrics",
				"GET /api/v1/metrics/refresh",
			},
		})
	})

	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"uptime_s": int64(time.Since(s.start).Seconds()),
			"time":     time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/api/v1/metrics", func(c *gin.Context) {
		res := s.sched.Last()
		if res == nil {
			c.JSON(http.StatusOK, gin.H{"message": "首轮采集尚未完成，请稍后重试"})
			return
		}
		c.JSON(http.StatusOK, res)
	})

	r.GET("/api/v1/metrics/refresh", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		c.JSON(http.StatusOK, s.sched.RunOnce(ctx))
	})

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})
}

// requestLogger 统一把访问日志写入项目日志器。
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t0 := time.Now()
		c.Next()
		logger.Debugf("http %s %s -> %d (%dms)",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(t0).Milliseconds())
	}
}

// ListenAndServe 启动服务。
func (s *Server) ListenAndServe() error {
	logger.Infof("HTTP 服务启动（gin），监听 %s", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 优雅关闭服务。
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
