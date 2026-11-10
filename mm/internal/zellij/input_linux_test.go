package zellij

import "testing"

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
