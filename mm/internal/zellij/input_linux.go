package zellij

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// readProcs lists every process from /proc.
func readProcs() []proc {
	entries, _ := os.ReadDir("/proc")
	var procs []proc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		dir := filepath.Join("/proc", e.Name())
		stat, err := os.ReadFile(filepath.Join(dir, "stat"))
		if err != nil {
			continue
		}
		ppid, ttyNr, ok := parseStat(string(stat))
		if !ok {
			continue
		}
		raw, _ := os.ReadFile(filepath.Join(dir, "cmdline"))
		var cmdline []string
		for arg := range bytes.SplitSeq(bytes.TrimRight(raw, "\x00"), []byte{0}) {
			cmdline = append(cmdline, string(arg))
		}
		tty, _ := ptsPath(ttyNr)
		procs = append(procs, proc{pid: pid, ppid: ppid, tty: tty, cmdline: cmdline})
	}
	return procs
}

// parseStat pulls ppid and tty_nr out of /proc/PID/stat. The command name in
// parentheses can hold spaces and parens, so fields count from the last ')'.
func parseStat(stat string) (ppid, ttyNr int, ok bool) {
	i := strings.LastIndexByte(stat, ')')
	if i < 0 {
		return 0, 0, false
	}
	// state ppid pgrp session tty_nr ...
	f := strings.Fields(stat[i+1:])
	if len(f) < 5 {
		return 0, 0, false
	}
	ppid, err1 := strconv.Atoi(f[1])
	ttyNr, err2 := strconv.Atoi(f[4])
	return ppid, ttyNr, err1 == nil && err2 == nil
}

// ptsPath decodes a tty_nr into /dev/pts/N. Unix98 pty slaves use majors
// 136-143, 256 minors each.
func ptsPath(ttyNr int) (string, bool) {
	major := (ttyNr >> 8) & 0xfff
	minor := (ttyNr & 0xff) | ((ttyNr >> 12) & 0xfff00)
	if major < 136 || major > 143 {
		return "", false
	}
	return fmt.Sprintf("/dev/pts/%d", (major-136)*256+minor), true
}

func ttyAtime(tty string) (int64, bool) {
	var st syscall.Stat_t
	if syscall.Stat(tty, &st) != nil {
		return 0, false
	}
	return st.Atim.Sec, true
}
