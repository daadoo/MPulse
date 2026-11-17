package collector

import (
	"context"
	"sort"

	"github.com/daadoo/MPulse/internal/model"
)

// ProcTopCollector 采集 CPU / 内存 Top N 高占用进程。
type ProcTopCollector struct{}

// Name 实现 Collector 接口。
func (ProcTopCollector) Name() string { return "proc_top" }

// Collect 实现 Collector 接口。
func (p ProcTopCollector) Collect(ctx context.Context, opt *Options, res *model.MetricResult) error {
	items := GetSnapshot(ctx, opt.IgnoreProc).Items

	byCPU := make([]model.ProcessItem, len(items))
	byMem := make([]model.ProcessItem, len(items))
	copy(byCPU, items)
	copy(byMem, items)

	sort.SliceStable(byCPU, func(i, j int) bool {
		if byCPU[i].CPUPercent == byCPU[j].CPUPercent {
			return byCPU[i].MemRSS > byCPU[j].MemRSS
		}
		return byCPU[i].CPUPercent > byCPU[j].CPUPercent
	})
	sort.SliceStable(byMem, func(i, j int) bool {
		if byMem[i].MemRSS == byMem[j].MemRSS {
			return byMem[i].CPUPercent > byMem[j].CPUPercent
		}
		return byMem[i].MemRSS > byMem[j].MemRSS
	})

	if opt.TopN < len(byCPU) {
		byCPU = byCPU[:opt.TopN]
	}
	if opt.TopN < len(byMem) {
		byMem = byMem[:opt.TopN]
	}

	res.TopCPU = byCPU
	res.TopMem = byMem
	return nil
}
