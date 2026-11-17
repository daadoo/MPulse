package collector

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/daadoo/MPulse/internal/logger"
	"github.com/daadoo/MPulse/internal/model"
)

// HostCollector 采集主机 CPU、内存、交换区与磁盘分区使用率。
type HostCollector struct{}

// Name 实现 Collector 接口。
func (HostCollector) Name() string { return "host" }

// Collect 实现 Collector 接口。
func (h HostCollector) Collect(ctx context.Context, opt *Options, res *model.MetricResult) error {
	info := &model.HostInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	t0 := time.Now()
	if hst, err := host.InfoWithContext(ctx); err == nil && hst != nil {
		info.Hostname = hst.Hostname
		info.OS = hst.Platform + " " + hst.PlatformVersion
		info.Uptime = hst.Uptime
	}
	logger.Debugf("host: 基本信息 %dms", time.Since(t0).Milliseconds())

	// CPU 使用率：采样 200ms 得到相对准确的值。
	t0 = time.Now()
	if pcts, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false); err == nil && len(pcts) > 0 {
		info.CPUPercent = round2(pcts[0])
	}
	logger.Debugf("host: CPU 采样 %dms", time.Since(t0).Milliseconds())

	t0 = time.Now()
	if runtime.GOOS == "linux" {
		if avg, err := load.AvgWithContext(ctx); err == nil && avg != nil {
			info.LoadAvg = []float64{round2(avg.Load1), round2(avg.Load5), round2(avg.Load15)}
		}
	}
	logger.Debugf("host: 负载 %dms", time.Since(t0).Milliseconds())

	t0 = time.Now()
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil && vm != nil {
		info.MemTotal = vm.Total
		info.MemUsed = vm.Used
		info.MemPercent = round2(vm.UsedPercent)
	}

	// Windows 的 SwapMemory 走 WMI 查询，单次开销可达 2s 以上，监控自身成本过高，故跳过。
	// Linux/macOS 读取 /proc、sysctl，成本可忽略。
	if runtime.GOOS != "windows" {
		if sm, err := mem.SwapMemoryWithContext(ctx); err == nil && sm != nil {
			info.SwapTotal = sm.Total
			info.SwapUsed = sm.Used
		}
	}
	logger.Debugf("host: 内存 %dms", time.Since(t0).Milliseconds())

	t0 = time.Now()
	parts, err := disk.PartitionsWithContext(ctx, false)
	logger.Debugf("host: 分区枚举 %dms (err=%v)", time.Since(t0).Milliseconds(), err)
	if err == nil {
		// 分区使用率查询在 Windows 上单次可达数百毫秒，按分区并行执行。
		type partJob struct {
			p     disk.PartitionStat
			usage *disk.UsageStat
		}
		jobs := make([]partJob, 0, len(parts))
		for _, p := range parts {
			if isIgnoredMount(p.Mountpoint, opt.IgnoreDisk) {
				continue
			}
			// 跳过常见虚拟/只读文件系统，避免噪音。
			if isVirtualFS(p.Fstype) {
				continue
			}
			jobs = append(jobs, partJob{p: p})
		}

		var wg sync.WaitGroup
		for i := range jobs {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				if u, err := disk.UsageWithContext(ctx, jobs[i].p.Mountpoint); err == nil && u != nil {
					jobs[i].usage = u
				}
			}()
		}
		wg.Wait()

		for _, j := range jobs {
			if j.usage == nil || j.usage.Total == 0 {
				continue
			}
			info.Disks = append(info.Disks, model.DiskInfo{
				Device:      j.p.Device,
				Mountpoint:  j.p.Mountpoint,
				Fstype:      j.p.Fstype,
				Total:       j.usage.Total,
				Used:        j.usage.Used,
				UsedPercent: round2(j.usage.UsedPercent),
			})
		}
	}

	res.Host = info
	return nil
}

// isIgnoredMount 判断挂载点是否在忽略列表。
func isIgnoredMount(mount string, ignore []string) bool {
	m := strings.TrimRight(mount, `/\`)
	for _, ig := range ignore {
		if strings.TrimRight(ig, `/\`) == m {
			return true
		}
	}
	return false
}

// isVirtualFS 判断是否为虚拟或只读文件系统。
func isVirtualFS(fstype string) bool {
	switch strings.ToLower(fstype) {
	case "squashfs", "tmpfs", "devtmpfs", "overlay", "proc", "sysfs", "cgroup", "cgroup2",
		"autofs", "debugfs", "tracefs", "ramfs", "nsfs", "bpf", "mqueue", "hugetlbfs":
		return true
	}
	return false
}
