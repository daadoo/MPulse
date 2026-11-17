// Package logger 提供轻量日志，支持控制台与文件双输出。
package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level 日志级别。
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// Logger 线程安全的日志器。
type Logger struct {
	mu    sync.Mutex
	level Level
	w     io.Writer
	file  *os.File
}

var std = &Logger{level: LevelInfo, w: os.Stdout}

// ParseLevel 解析日志级别字符串，非法值回落到 info。
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Setup 初始化全局日志器，filePath 为空则只输出到控制台。
func Setup(level, filePath string) error {
	std.mu.Lock()
	defer std.mu.Unlock()
	std.level = ParseLevel(level)
	if filePath == "" {
		return nil
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	std.file = f
	std.w = io.MultiWriter(os.Stdout, f)
	return nil
}

// Close 关闭日志文件。
func Close() {
	std.mu.Lock()
	defer std.mu.Unlock()
	if std.file != nil {
		_ = std.file.Close()
		std.file = nil
		std.w = os.Stdout
	}
}

func (l *Logger) log(lv Level, format string, args ...any) {
	if lv < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.w, "%s [%s] %s\n", ts, levelNames[lv], msg)
}

// Debugf 输出 debug 日志。
func Debugf(format string, args ...any) { std.log(LevelDebug, format, args...) }

// Infof 输出 info 日志。
func Infof(format string, args ...any) { std.log(LevelInfo, format, args...) }

// Warnf 输出 warn 日志。
func Warnf(format string, args ...any) { std.log(LevelWarn, format, args...) }

// Errorf 输出 error 日志。
func Errorf(format string, args ...any) { std.log(LevelError, format, args...) }
