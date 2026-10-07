package zellij

import (
	"reflect"
	"testing"
)

func TestParseSessions(t *testing.T) {
	out := "hop [Created 4months 21days 23h 51m 25s ago] (EXITED - attach to resurrect)\n" +
		"mason [Created 1day 6h 27m 45s ago] (current)\n" +
		"\n" +
		"fresh\n"
	want := []Session{
		{Name: "hop", Exited: true, Created: 4*2592000 + 21*86400 + 23*3600 + 51*60 + 25},
		{Name: "mason", Created: 86400 + 6*3600 + 27*60 + 45},
		{Name: "fresh"},
	}
	if got := ParseSessions(out); !reflect.DeepEqual(got, want) {
		t.Errorf("ParseSessions = %+v, want %+v", got, want)
	}
}

func TestParseLayoutCWD(t *testing.T) {
	layout := "layout {\n    cwd \"/home/me/Code\"\n    tab name=\"x\" {\n        pane cwd=\"sub\"\n        cwd \"/elsewhere\"\n    }\n}\n"
	if got := ParseLayoutCWD(layout); got != "/home/me/Code" {
		t.Errorf("ParseLayoutCWD = %q", got)
	}
	if got := ParseLayoutCWD("layout {\n}\n"); got != "" {
		t.Errorf("ParseLayoutCWD without cwd = %q", got)
	}
}

func TestParseStat(t *testing.T) {
	ppid, tty, ok := parseStat("1234 (my (odd) proc) S 99 1234 1234 34817 1234 4194560")
	if !ok || ppid != 99 || tty != 34817 {
		t.Errorf("parseStat = %d, %d, %v", ppid, tty, ok)
	}
	if _, _, ok := parseStat("garbage"); ok {
		t.Error("parseStat accepted garbage")
	}
}

func TestPtsPath(t *testing.T) {
	// 34817 is major 136, minor 1.
	if got, ok := ptsPath(34817); !ok || got != "/dev/pts/1" {
		t.Errorf("ptsPath(34817) = %q, %v", got, ok)
	}
	// Major 137, minor 4 is the 260th pty.
	if got, ok := ptsPath(137<<8 | 4); !ok || got != "/dev/pts/260" {
		t.Errorf("ptsPath(137:4) = %q, %v", got, ok)
	}
	if _, ok := ptsPath(0); ok {
		t.Error("ptsPath(0) is no tty")
	}
}

func TestServerSession(t *testing.T) {
	name, ok := serverSession([]string{"/home/me/bin/zellij", "--server", "/run/user/1000/zellij/0.45.1/che"})
	if !ok || name != "che" {
		t.Errorf("serverSession = %q, %v", name, ok)
	}
	if _, ok := serverSession([]string{"zellij", "attach", "che"}); ok {
		t.Error("serverSession matched a client")
	}
	if _, ok := serverSession([]string{"grep", "--server", "x"}); ok {
		t.Error("serverSession matched a non-zellij process")
	}
}
