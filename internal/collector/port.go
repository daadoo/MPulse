package collector

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/net"
	psproc "github.com/shirou/gopsutil/v4/process"

	"github.com/daadoo/MPulse/internal/model"
)

// PortCollector 采集指定端口的监听状态与占用进程资源。
type PortCollector struct {
	ports []int
}

// NewPortCollector 构造端口采集器。
func NewPortCollector(ports []int) *PortCollector {
	return &PortCollector{ports: ports}
}

// Name 实现 Collector 接口。
func (p *PortCollector) Name() string { return "port" }

// Collect 实现 Collector 接口。
func (p *PortCollector) Collect(ctx context.Context, opt *Options, res *model.MetricResult) error {
	if len(p.ports) == 0 {
		return nil
	}

	conns, err := net.ConnectionsWithContext(ctx, "tcp")
	if err != nil {
		return fmt.Errorf("读取 TCP 连接表失败: %w", err)
	}

	// 建立 端口 -> pid 的监听索引。
	owner := make(map[int]int32, len(p.ports))
	for _, c := range conns {
		if c.Status != "LISTEN" && c.Status != "listen" {
			continue
		}
		if _, ok := owner[int(c.Laddr.Port)]; !ok && c.Pid > 0 {
			owner[int(c.Laddr.Port)] = c.Pid
		}
	}

	for _, port := range p.ports {
		usage := model.PortUsage{Port: port}
		pid, ok := owner[port]
		if !ok {
			res.Ports = append(res.Ports, usage)
			continue
		}
		usage.Listening = true
		usage.PID = pid

		proc, err := psproc.NewProcess(pid)
		if err != nil {
			usage.Error = err.Error()
			res.Ports = append(res.Ports, usage)
			continue
		}
		if name, err := proc.Name(); err == nil {
			usage.Name = name
		}
		if cpu, err := proc.CPUPercent(); err == nil {
			usage.CPUPercent = round2(cpu)
		}
		if mi, err := proc.MemoryInfo(); err == nil && mi != nil {
			usage.MemRSS = mi.RSS
		}
		if pct, err := proc.MemoryPercent(); err == nil {
			usage.MemPercent = round2(float64(pct))
		}
		res.Ports = append(res.Ports, usage)
	}
	return nil
}
