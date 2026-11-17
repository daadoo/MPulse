package common

import "testing"

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}
	for _, c := range cases {
		if got := FormatBytes(c.in); got != c.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplitList(t *testing.T) {
	got := SplitList(" nginx, MySQL ,nginx, ,redis ")
	want := []string{"nginx", "MySQL", "redis"}
	if len(got) != len(want) {
		t.Fatalf("SplitList 长度 = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SplitList[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if SplitList("   ") != nil {
		t.Error("空白字符串应返回 nil")
	}
}

func TestNormalizeProcName(t *testing.T) {
	if got := NormalizeProcName("  NGINX.EXE "); got != "nginx" {
		t.Errorf("NormalizeProcName = %q, want nginx", got)
	}
}

func TestContainsFold(t *testing.T) {
	list := []string{"mysqld", "Nginx"}
	if !ContainsFold(list, "NGINX.exe") {
		t.Error("应命中 Nginx")
	}
	if ContainsFold(list, "redis") {
		t.Error("不应命中 redis")
	}
}
