package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.Mode != "cli" {
		t.Errorf("默认 mode = %q, want cli", c.Mode)
	}
	if c.Interval != 30 || c.TopN != 10 || c.Listen != ":55055" {
		t.Errorf("默认值异常: %+v", c)
	}
}

func TestLoadFlagsOverride(t *testing.T) {
	c, err := Load([]string{"-mode", "daemon", "-interval", "5", "-ports", "80,443,abc,70000", "-top", "3"})
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if c.Mode != "daemon" || c.Interval != 5 || c.TopN != 3 {
		t.Errorf("参数未生效: %+v", c)
	}
	if len(c.Ports) != 2 || c.Ports[0] != 80 || c.Ports[1] != 443 {
		t.Errorf("端口解析异常（应忽略非法项）: %v", c.Ports)
	}
}

func TestLoadInvalid(t *testing.T) {
	if _, err := Load([]string{"-mode", "xxx"}); err == nil {
		t.Error("非法 mode 应返回错误")
	}
	if _, err := Load([]string{"-interval", "0"}); err == nil {
		t.Error("interval=0 应返回错误")
	}
}

func TestLoadYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	content := "mode: daemon\ninterval: 15\nlisten: \":8080\"\ntop_n: 5\nports:\n  - 22\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load([]string{"-config", path})
	if err != nil {
		t.Fatalf("加载 YAML 失败: %v", err)
	}
	if c.Mode != "daemon" || c.Interval != 15 || c.Listen != ":8080" || c.TopN != 5 || len(c.Ports) != 1 {
		t.Errorf("YAML 未生效: %+v", c)
	}
}

func TestFlagsOverrideFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.json")
	if err := os.WriteFile(path, []byte(`{"mode":"daemon","interval":15}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load([]string{"-config", path, "-interval", "60"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Mode != "daemon" {
		t.Errorf("配置文件 mode 应生效, got %q", c.Mode)
	}
	if c.Interval != 60 {
		t.Errorf("命令行应覆盖配置文件, got %d", c.Interval)
	}
}
