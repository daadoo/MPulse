package collector

import (
	"context"
	"strings"

	"MPulse/internal/model"
)

// GroupCollector 按进程名匹配一组进程并汇总资源消耗（nginx / 数据库 / 自定义服务）。
type GroupCollector struct {
	group    string
	matchers []string
}

// NewNginxCollector 构造 Nginx 进程汇总采集器。
func NewNginxCollector(names []string) *GroupCollector {
	return &GroupCollector{group: "nginx", matchers: names}
}

// NewDBCollector 构造数据库进程汇总采集器。
func NewDBCollector(names []string) *GroupCollector {
	return &GroupCollector{group: "database", matchers: names}
}

// NewServiceCollector 构造自定义服务采集器。
func NewServiceCollector(name string) *GroupCollector {
	return &GroupCollector{group: "service:" + name, matchers: []string{name}}
}

// Name 实现 Collector 接口。
func (g *GroupCollector) Name() string { return "group_" + g.group }

// Collect 实现 Collector 接口。
func (g *GroupCollector) Collect(ctx context.Context, opt *Options, res *model.MetricResult) error {
	if len(g.matchers) == 0 {
		return nil
	}
	items := GetSnapshot(ctx, opt.IgnoreProc).Items

	usage := model.GroupUsage{Group: g.group, Matched: g.matchers}
	for _, it := range items {
		if !matchProcName(it.Name, g.matchers) {
			continue
		}
		usage.Count++
		usage.CPUPercent += it.CPUPercent
		usage.MemRSS += it.MemRSS
		usage.MemPercent += it.MemPercent
		usage.Processes = append(usage.Processes, it)
	}
	usage.CPUPercent = round2(usage.CPUPercent)
	usage.MemPercent = round2(usage.MemPercent)

	res.Groups = append(res.Groups, usage)
	return nil
}

// matchProcName 判断进程名是否命中匹配列表（忽略大小写与 .exe 后缀）。
func matchProcName(name string, matchers []string) bool {
	n := strings.ToLower(strings.TrimSuffix(name, ".exe"))
	for _, m := range matchers {
		mm := strings.ToLower(strings.TrimSuffix(m, ".exe"))
		if mm == n || strings.HasPrefix(n, mm+"-") || strings.HasPrefix(n, mm+"worker") {
			return true
		}
	}
	return false
}
