// Package model 定义 MPulse 全量采集结果的公共数据结构。
package model

// HostInfo 主机基础指标：CPU、内存、交换区、磁盘分区。
type HostInfo struct {
	Hostname   string     `json:"hostname"`
	OS         string     `json:"os"`
	Arch       string     `json:"arch"`
	CPUCores   int        `json:"cpu_cores"`
	CPUPercent float64    `json:"cpu_percent"`
	LoadAvg    []float64  `json:"load_avg,omitempty"`
	MemTotal   uint64     `json:"mem_total"`
	MemUsed    uint64     `json:"mem_used"`
	MemPercent float64    `json:"mem_percent"`
	SwapTotal  uint64     `json:"swap_total"`
	SwapUsed   uint64     `json:"swap_used"`
	Uptime     uint64     `json:"uptime"`
	Disks      []DiskInfo `json:"disks"`
}

// DiskInfo 单个磁盘分区的使用情况。
type DiskInfo struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

// ProcessItem 单个进程的资源占用快照。
type ProcessItem struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	Cmdline    string  `json:"cmdline,omitempty"`
	CPUPercent float64 `json:"cpu_percent"`
	MemRSS     uint64  `json:"mem_rss"`
	MemPercent float64 `json:"mem_percent"`
}

// GroupUsage 一组同类进程（如 nginx、数据库）的汇总消耗。
type GroupUsage struct {
	Group      string        `json:"group"`
	Matched    []string      `json:"matched"`
	Count      int           `json:"count"`
	CPUPercent float64       `json:"cpu_percent"`
	MemRSS     uint64        `json:"mem_rss"`
	MemPercent float64       `json:"mem_percent"`
	Processes  []ProcessItem `json:"processes"`
}

// PortUsage 指定端口的监听状态及占用进程资源。
type PortUsage struct {
	Port       int     `json:"port"`
	Listening  bool    `json:"listening"`
	PID        int32   `json:"pid"`
	Name       string  `json:"name,omitempty"`
	CPUPercent float64 `json:"cpu_percent"`
	MemRSS     uint64  `json:"mem_rss"`
	MemPercent float64 `json:"mem_percent"`
	Error      string  `json:"error,omitempty"`
}

// ErrorInfo 单个采集器的失败信息，采集失败不阻断整轮采集。
type ErrorInfo struct {
	Collector string `json:"collector"`
	Error     string `json:"error"`
}

// MetricResult 一轮采集的完整结果。
type MetricResult struct {
	Timestamp  string        `json:"timestamp"`
	DurationMS int64         `json:"duration_ms"`
	Host       *HostInfo     `json:"host,omitempty"`
	TopCPU     []ProcessItem `json:"top_cpu"`
	TopMem     []ProcessItem `json:"top_mem"`
	Groups     []GroupUsage  `json:"groups"`
	Ports      []PortUsage   `json:"ports"`
	Errors     []ErrorInfo   `json:"errors,omitempty"`
}
