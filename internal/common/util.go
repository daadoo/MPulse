// Package common 提供常量、单位换算与字符串处理等公共工具。
package common

import (
	"fmt"
	"strings"
)

const (
	// Version 默认版本号，构建时可用 -ldflags 覆盖。
	Version = "1.0.0"
	// DefaultListen 默认 Web 监听地址。
	DefaultListen = ":55055"
	// DefaultInterval 默认采集间隔（秒）。
	DefaultInterval = 30
	// DefaultTopN 默认 Top 进程数量。
	DefaultTopN = 10
	// ProcCPUSampleWindow 进程 CPU 采样窗口（毫秒）。
	ProcCPUSampleWindow = 150
)

// FormatBytes 将字节数转换为易读单位（B/KB/MB/GB/TB）。
func FormatBytes(v uint64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%d B", v)
	}
	div, exp := uint64(unit), 0
	for n := v / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(v)/float64(div), "KMGTP"[exp])
}

// SplitList 将逗号分隔的字符串切分为去空、去重的切片。
func SplitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := NormalizeProcName(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

// ContainsFold 判断切片中是否包含目标进程名（忽略大小写与 .exe 后缀）。
func ContainsFold(list []string, target string) bool {
	t := NormalizeProcName(target)
	for _, item := range list {
		if NormalizeProcName(item) == t {
			return true
		}
	}
	return false
}

// NormalizeProcName 统一进程名：转小写并去掉 .exe 后缀。
func NormalizeProcName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	return strings.TrimSuffix(n, ".exe")
}
