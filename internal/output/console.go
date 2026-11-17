// Package output 负责采集结果的落地输出：控制台、日志、HTTP JSON API。
package output

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/daadoo/MPulse/internal/common"
	"github.com/daadoo/MPulse/internal/model"
)

// Console 将采集结果以表格形式写入 w。
func Console(w io.Writer, res *model.MetricResult) error {
	if res == nil {
		_, err := fmt.Fprintln(w, "无采集结果")
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	fmt.Fprintf(tw, "采集时间\t%s\t耗时\t%dms\n", res.Timestamp, res.DurationMS)
	if h := res.Host; h != nil {
		fmt.Fprintf(tw, "主机\t%s (%s/%s)\n", h.Hostname, h.OS, h.Arch)
		fmt.Fprintf(tw, "CPU\t%.2f%%\t核心\t%d\n", h.CPUPercent, h.CPUCores)
		fmt.Fprintf(tw, "内存\t%s / %s (%.2f%%)\n", common.FormatBytes(h.MemUsed), common.FormatBytes(h.MemTotal), h.MemPercent)
		if h.SwapTotal > 0 {
			fmt.Fprintf(tw, "交换区\t%s / %s\n", common.FormatBytes(h.SwapUsed), common.FormatBytes(h.SwapTotal))
		}
		for _, d := range h.Disks {
			fmt.Fprintf(tw, "磁盘\t%s\t%s\t%s / %s (%.2f%%)\n",
				d.Mountpoint, d.Fstype, common.FormatBytes(d.Used), common.FormatBytes(d.Total), d.UsedPercent)
		}
	}

	printProcs(tw, "CPU Top", res.TopCPU)
	printProcs(tw, "内存 Top", res.TopMem)

	for _, g := range res.Groups {
		fmt.Fprintf(tw, "\n[%s] 进程数\t%d\tCPU\t%.2f%%\t内存\t%s (%.2f%%)\n",
			g.Group, g.Count, g.CPUPercent, common.FormatBytes(g.MemRSS), g.MemPercent)
	}

	if len(res.Ports) > 0 {
		fmt.Fprintf(tw, "\n端口\t监听\tPID\t进程\tCPU\t内存\n")
		for _, p := range res.Ports {
			if !p.Listening {
				fmt.Fprintf(tw, "%d\t否\t-\t-\t-\t-\n", p.Port)
				continue
			}
			fmt.Fprintf(tw, "%d\t是\t%d\t%s\t%.2f%%\t%s\n",
				p.Port, p.PID, p.Name, p.CPUPercent, common.FormatBytes(p.MemRSS))
		}
	}

	if len(res.Errors) > 0 {
		fmt.Fprintf(tw, "\n采集错误\n")
		for _, e := range res.Errors {
			fmt.Fprintf(tw, "%s\t%s\n", e.Collector, e.Error)
		}
	}
	return tw.Flush()
}

func printProcs(tw *tabwriter.Writer, title string, items []model.ProcessItem) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(tw, "\n%s\nPID\t进程\tCPU\t内存\t占比\n", title)
	for _, p := range items {
		fmt.Fprintf(tw, "%d\t%s\t%.2f%%\t%s\t%.2f%%\n",
			p.PID, p.Name, p.CPUPercent, common.FormatBytes(p.MemRSS), p.MemPercent)
	}
}

// LogLine 生成单行日志摘要。
func LogLine(res *model.MetricResult) string {
	if res == nil || res.Host == nil {
		return "无采集结果"
	}
	h := res.Host
	line := fmt.Sprintf("cpu=%.2f%% mem=%s/%s(%.2f%%)",
		h.CPUPercent, common.FormatBytes(h.MemUsed), common.FormatBytes(h.MemTotal), h.MemPercent)
	for _, g := range res.Groups {
		line += fmt.Sprintf(" %s=%d个/%.2f%%/ %s", g.Group, g.Count, g.CPUPercent, common.FormatBytes(g.MemRSS))
	}
	return fmt.Sprintf("%s 耗时=%dms", line, res.DurationMS)
}

// TimeNow 便于其它包统一时间格式。
func TimeNow() string { return time.Now().Format("2006-01-02 15:04:05") }
