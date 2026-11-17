// Package config 负责配置解析，优先级：命令行参数 > 配置文件 > 默认值。
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/daadoo/MPulse/internal/common"
)

// Config 运行配置。
type Config struct {
	Mode           string   `json:"mode" yaml:"mode"`         //cli or daemon，缺省cli
	Interval       int      `json:"interval" yaml:"interval"` //模式采集间隔，如果<=0，只能是cli模式，仅采集查询一次
	Listen         string   `json:"listen" yaml:"listen"`     //查询服务端口
	LogLevel       string   `json:"log_level" yaml:"log_level"`
	LogFile        string   `json:"log_file" yaml:"log_file"`
	TopN           int      `json:"top_n" yaml:"top_n"`
	Ports          []int    `json:"ports" yaml:"ports"`
	Services       []string `json:"services" yaml:"services"`
	DBProcesses    []string `json:"db_processes" yaml:"db_processes"`
	NginxProcesses []string `json:"nginx_processes" yaml:"nginx_processes"`
	IgnoreProc     []string `json:"ignore_proc" yaml:"ignore_proc"`
	IgnoreDisk     []string `json:"ignore_disk" yaml:"ignore_disk"`

	// ShowVersion 由 -version 触发，调用方负责打印后退出。
	ShowVersion bool `json:"-" yaml:"-"`
	// ShowHelp   负责使用帮助信息，没有有价值的输入参数，显示该信息。
	ShowHelp bool `json:"-" yaml:"-"`

	// ConfigPath 实际加载的配置文件路径，仅用于日志展示。
	ConfigPath string `json:"-" yaml:"-"`
}

// Default 返回默认配置。
func Default() *Config {
	return &Config{
		Mode:           "cli",
		Interval:       common.DefaultInterval,
		Listen:         common.DefaultListen,
		LogLevel:       "info",
		TopN:           common.DefaultTopN,
		DBProcesses:    []string{"mysqld", "mariadbd", "postgres", "mongod", "redis-server"},
		NginxProcesses: []string{"nginx"},
	}
}

// Load 解析命令行参数与配置文件，返回最终配置。
func Load(args []string) (*Config, error) {
	cfg := Default()
	// 使用flag标准库解析命令行参数。
	fs := flag.NewFlagSet("mpulse", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		mode        = fs.String("mode", cfg.Mode, "运行模式：cli 一次性采集 / daemon 常驻守护")
		interval    = fs.Int("interval", cfg.Interval, "daemon 模式采集间隔（秒）")
		listen      = fs.String("listen", cfg.Listen, "HTTP 监听地址，如 :55055")
		logLevel    = fs.String("log-level", cfg.LogLevel, "日志级别：debug/info/warn/error")
		logFile     = fs.String("log-file", cfg.LogFile, "日志文件路径，为空则仅输出到控制台")
		topN        = fs.Int("top", cfg.TopN, "Top 高占用进程数量")
		ports       = fs.String("ports", "", "需要观测的端口列表，逗号分隔，如 80,443,3306")
		services    = fs.String("services", "", "需要观测的自定义服务进程名，逗号分隔")
		dbProcs     = fs.String("db-procs", strings.Join(cfg.DBProcesses, ","), "数据库进程名列表，逗号分隔")
		nginxProcs  = fs.String("nginx-procs", strings.Join(cfg.NginxProcesses, ","), "Nginx 进程名列表，逗号分隔")
		ignoreProc  = fs.String("ignore-proc", "", "忽略的进程名列表，逗号分隔")
		ignoreDisk  = fs.String("ignore-disk", "", "忽略的磁盘挂载点列表，逗号分隔")
		configPath  = fs.String("config", "", "配置文件路径，支持 .json/.yaml/.yml")
		showVersion = fs.Bool("version", false, "打印版本号后退出")
		showHelp    = fs.Bool("help", false, "打印使用帮助")
	)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.ShowHelp = *showHelp
	if cfg.ShowHelp == true || len(args) == 0 {
		cfg.ShowHelp = true
		return cfg, nil
	}

	// 先加载配置文件，再用显式传入的命令行参数覆盖。
	if *configPath != "" {
		if err := loadFile(cfg, *configPath); err != nil {
			return nil, err
		}
		cfg.ConfigPath = *configPath
	}

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })

	if setFlags["mode"] {
		cfg.Mode = *mode
	}
	if setFlags["interval"] {
		cfg.Interval = *interval
	}
	if setFlags["listen"] {
		cfg.Listen = *listen
	}
	if setFlags["log-level"] {
		cfg.LogLevel = *logLevel
	}
	if setFlags["log-file"] {
		cfg.LogFile = *logFile
	}
	if setFlags["top"] {
		cfg.TopN = *topN
	}
	if setFlags["ports"] {
		cfg.Ports = parsePorts(*ports)
	}
	if setFlags["services"] {
		cfg.Services = append(cfg.Services, common.SplitList(*services)...)
	}
	if setFlags["db-procs"] {
		cfg.DBProcesses = common.SplitList(*dbProcs)
	}
	if setFlags["nginx-procs"] {
		cfg.NginxProcesses = common.SplitList(*nginxProcs)
	}
	if setFlags["ignore-proc"] {
		cfg.IgnoreProc = common.SplitList(*ignoreProc)
	}
	if setFlags["ignore-disk"] {
		cfg.IgnoreDisk = common.SplitList(*ignoreDisk)
	}
	cfg.ShowVersion = *showVersion

	if cfg.Mode != "cli" && cfg.Mode != "daemon" {
		return nil, fmt.Errorf("非法 mode %q，可选 cli / daemon", cfg.Mode)
	}
	if cfg.Interval <= 0 {
		return nil, fmt.Errorf("interval 必须大于 0，当前 %d", cfg.Interval)
	}
	if cfg.TopN <= 0 {
		return nil, fmt.Errorf("top 必须大于 0，当前 %d", cfg.TopN)
	}
	return cfg, nil
}

// loadFile 读取 JSON 或 YAML 配置文件并覆盖默认值。
func loadFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("解析 YAML 配置失败: %w", err)
		}
		return nil
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("解析 JSON 配置失败: %w", err)
	}
	return nil
}

// parsePorts 解析端口列表，非法项忽略。
func parsePorts(s string) []int {
	var out []int
	for _, p := range common.SplitList(s) {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 || n > 65535 {
			continue
		}
		out = append(out, n)
	}
	return out
}
