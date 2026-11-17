// Package scheduler 负责采集调度：一轮采集内的采集器并行执行，守护模式下定时循环。
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/daadoo/MPulse/internal/collector"
	"github.com/daadoo/MPulse/internal/common"
	"github.com/daadoo/MPulse/internal/config"
	"github.com/daadoo/MPulse/internal/logger"
	"github.com/daadoo/MPulse/internal/model"
)

// Scheduler 采集调度器。
type Scheduler struct {
	cfg        *config.Config
	collectors []collector.Collector

	mu   sync.RWMutex
	last *model.MetricResult
}

// New 根据配置构建调度器与采集器组。
func New(cfg *config.Config) *Scheduler {
	opt := &collector.Options{
		TopN:           cfg.TopN,
		Ports:          cfg.Ports,
		Services:       cfg.Services,
		DBProcesses:    cfg.DBProcesses,
		NginxProcesses: cfg.NginxProcesses,
		IgnoreProc:     cfg.IgnoreProc,
		IgnoreDisk:     cfg.IgnoreDisk,
	}

	s := &Scheduler{cfg: cfg}
	s.collectors = []collector.Collector{
		collector.HostCollector{},
		collector.ProcTopCollector{},
		collector.NewNginxCollector(opt.NginxProcesses),
		collector.NewDBCollector(opt.DBProcesses),
	}
	for _, svc := range opt.Services {
		s.collectors = append(s.collectors, collector.NewServiceCollector(svc))
	}
	if len(opt.Ports) > 0 {
		s.collectors = append(s.collectors, collector.NewPortCollector(opt.Ports))
	}
	return s
}

// Options 返回由配置转换的采集参数。
func (s *Scheduler) Options() *collector.Options {
	return &collector.Options{
		TopN:           s.cfg.TopN,
		Ports:          s.cfg.Ports,
		Services:       s.cfg.Services,
		DBProcesses:    s.cfg.DBProcesses,
		NginxProcesses: s.cfg.NginxProcesses,
		IgnoreProc:     s.cfg.IgnoreProc,
		IgnoreDisk:     s.cfg.IgnoreDisk,
	}
}

// RunOnce 执行一轮采集：进程快照先构建一次，其余采集器并行执行。
func (s *Scheduler) RunOnce(ctx context.Context) *model.MetricResult {
	start := time.Now()
	collector.ResetSnapshot()

	res := &model.MetricResult{
		Timestamp: start.Format(time.RFC3339),
	}

	// 先构建进程快照并预热 CPU 采样，其它采集器复用该快照。
	collector.GetSnapshot(ctx, s.cfg.IgnoreProc)

	var wg sync.WaitGroup
	var mu sync.Mutex
	opt := s.Options()

	for _, c := range s.collectors {
		c := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			t0 := time.Now()
			if err := c.Collect(ctx, opt, res); err != nil {
				mu.Lock()
				res.Errors = append(res.Errors, model.ErrorInfo{Collector: c.Name(), Error: err.Error()})
				mu.Unlock()
				logger.Warnf("采集器 %s 执行失败: %v", c.Name(), err)
			}
			logger.Debugf("采集器 %s 耗时 %dms", c.Name(), time.Since(t0).Milliseconds())
		}()
	}
	wg.Wait()

	res.DurationMS = time.Since(start).Milliseconds()
	s.setLast(res)
	return res
}

// Run 按配置间隔持续采集，ctx 取消时退出。
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(s.cfg.Interval) * time.Second)
	defer ticker.Stop()

	logger.Infof("守护模式启动，采集间隔 %ds，监听 %s", s.cfg.Interval, s.cfg.Listen)
	s.RunOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Infof("采集循环退出")
			return
		case <-ticker.C: //每个tick执行一次RunOnce即可
			s.RunOnce(ctx)
		}
	}
}

// Last 返回最近一轮采集结果，未采集时返回 nil。
func (s *Scheduler) Last() *model.MetricResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.last
}

func (s *Scheduler) setLast(res *model.MetricResult) {
	s.mu.Lock()
	s.last = res
	s.mu.Unlock()
}

// Summary 生成一行摘要文本，供日志输出。
func Summary(res *model.MetricResult) string {
	if res == nil || res.Host == nil {
		return "无采集结果"
	}
	h := res.Host
	return fmt.Sprintf("CPU %.2f%% 内存 %s/%s 耗时 %dms",
		h.CPUPercent, common.FormatBytes(h.MemUsed), common.FormatBytes(h.MemTotal), res.DurationMS)
}
