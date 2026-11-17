// MPulse 是一款服务器性能巡检工具。
// 操作支持CLI，支持一次性采集输出 与 Daemon常驻监控服务两种模式。
// Author: mengxin
// Create: 2026-11-17
// ChangeLog: 2026-11-17 创建项目工程
// file: main.go 项目主文件
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"MPulse/internal/config"
	"MPulse/internal/logger"
	"MPulse/internal/output"
	"MPulse/internal/scheduler"
)

var (
	version = "dev 1.0.0"
	commit  = "服务器运行状态查询工具"
	date    = "2026-11-17"
)

func main() {
	// 子命令在常规 flag 解析之前拦截：eval 是离线评测 harness，参数体系独立。
	if len(os.Args) > 1 && os.Args[1] == "eval" {
		os.Exit(1)
	}
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg, err := config.Load(args) //根据输入参数和配置文件，解析运行配置，输出到cfg对象中
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置解析失败: %v\n", err)
		return 2
	}
	if cfg.ShowVersion {
		//打印APP版本信息
		fmt.Printf("MPulse %s (commit %s, built %s)\n", version, commit, date)
		return 0
	}

	if cfg.ShowHelp {
		//打印帮助信息
		output.PrintHelp()
		return 0
	}

	if err := logger.Setup(cfg.LogLevel, cfg.LogFile); err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
		return 2
	}
	defer logger.Close()

	//进程收到 Ctrl+C / SIGTERM,优雅退出
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sched := scheduler.New(cfg)

	if cfg.Mode == "cli" { //单次查询模式
		res := sched.RunOnce(ctx)
		if err := output.Console(os.Stdout, res); err != nil {
			fmt.Fprintf(os.Stderr, "输出失败: %v\n", err)
			return 1
		}
		logger.Infof("CLI 采集完成: %s", output.LogLine(res))
		return 0
	}

	// daemon 监控服务模式：启动 HTTP 服务，随后进入定时采集循环，收到信号后优雅退出。
	srv := output.NewServer(cfg.Listen, sched)
	serverErr := make(chan error, 1)
	go func() { serverErr <- srv.ListenAndServe() }()

	sched.Run(ctx)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	if err := <-serverErr; err != nil {
		logger.Errorf("HTTP 服务异常退出: %v", err)
		return 1
	}
	return 0
}

// 命令行上打印显示help信息
func showHelp() {
	helpMsg := `MPulse: a server running status watch`
	fmt.Printf("%s\n", helpMsg)
	output.PrintHelp()
}
