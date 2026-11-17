// Package collector 定义采集器接口与进程快照缓存。
package collector

import (
	"context"
	"strings"
	"sync"
	"time"

	psproc "github.com/shirou/gopsutil/v4/process"

	"github.com/daadoo/MPulse/internal/common"
	"github.com/daadoo/MPulse/internal/model"
)

// Collector 单个指标采集器。
type Collector interface {
	// Name 采集器名称，用于错误归类。
	Name() string
	// Collect 执行采集，结果写入 res；返回错误由调度层收集，不阻断其它采集器。
	Collect(ctx context.Context, cfg *Options, res *model.MetricResult) error
}

// Options 采集参数，由配置层转换而来。
type Options struct {
	TopN           int
	Ports          []int
	Services       []string
	DBProcesses    []string
	NginxProcesses []string
	IgnoreProc     []string
	IgnoreDisk     []string
}

// ProcSnapshot 一轮采集共享的进程快照，避免各采集器重复遍历进程表。
type ProcSnapshot struct {
	Items []model.ProcessItem
}

var (
	snapMu   sync.Mutex
	snapOnce *sync.Once
	snapData *ProcSnapshot
)

// ResetSnapshot 在每轮采集开始时调用，使后续采集器复用同一份进程快照。
func ResetSnapshot() {
	snapMu.Lock()
	defer snapMu.Unlock()
	snapData = nil
	snapOnce = &sync.Once{}
}

// GetSnapshot 返回本轮采集的进程快照，首次调用时执行一次全量采样并阻塞至采样完成。
func GetSnapshot(ctx context.Context, ignore []string) *ProcSnapshot {
	snapMu.Lock()
	if snapOnce == nil {
		snapOnce = &sync.Once{}
	}
	once := snapOnce
	snapMu.Unlock()

	once.Do(func() {
		data := buildSnapshot(ctx, ignore)
		snapMu.Lock()
		snapData = data
		snapMu.Unlock()
	})

	snapMu.Lock()
	defer snapMu.Unlock()
	if snapData == nil {
		return &ProcSnapshot{}
	}
	return snapData
}

// buildSnapshot 两轮采样遍历进程表：第一轮预热 CPU 计时，第二轮读取采样窗口内的真实占用。
func buildSnapshot(ctx context.Context, ignore []string) *ProcSnapshot {
	pids, err := psproc.Pids()
	if err != nil {
		return &ProcSnapshot{}
	}

	procs := make([]*psproc.Process, 0, len(pids))
	names := make([]string, 0, len(pids))
	for _, pid := range pids {
		p, err := psproc.NewProcess(pid)
		if err != nil {
			continue
		}
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}
		if isIgnored(name, ignore) {
			continue
		}
		// 预热：记录当前 CPU 计时基准，第二轮据此计算窗口内增量。
		_, _ = p.CPUPercent()
		procs = append(procs, p)
		names = append(names, name)
	}

	// 采样窗口内等待，让 CPU 计时产生可测量的增量。
	select {
	case <-ctx.Done():
	case <-time.After(common.ProcCPUSampleWindow * time.Millisecond):
	}

	items := make([]model.ProcessItem, 0, len(procs))
	for i, p := range procs {
		if ctx.Err() != nil {
			break
		}
		it := model.ProcessItem{PID: p.Pid, Name: names[i]}
		if cpu, err := p.CPUPercent(); err == nil {
			it.CPUPercent = round2(cpu)
		}
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			it.MemRSS = mi.RSS
		}
		if pct, err := p.MemoryPercent(); err == nil {
			it.MemPercent = round2(float64(pct))
		}
		items = append(items, it)
	}
	return &ProcSnapshot{Items: items}
}

// isIgnored 判断进程是否在忽略列表中（忽略大小写与 .exe 后缀）。
func isIgnored(name string, ignore []string) bool {
	n := strings.ToLower(strings.TrimSuffix(name, ".exe"))
	for _, ig := range ignore {
		if strings.ToLower(strings.TrimSuffix(ig, ".exe")) == n {
			return true
		}
	}
	return false
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
