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

// LastInput maps each live session to when it was last typed in, as epoch
// seconds. Every pane is its own pty, and the kernel bumps a tty's atime
// whenever the program in it reads input -- the clock `w` uses for IDLE -- so
// the newest atime across a session's panes is its last keystroke. Output only
// bumps mtime, so a busy agent doesn't count as use. The panes are the
// server's direct children; anything started from a pane's shell shares its
// pty.
func LastInput() map[string]int64 {
	procs := readProcs()
	servers := map[int]string{}
	for _, p := range procs {
		if name, ok := serverSession(p.cmdline); ok {
			servers[p.pid] = name
		}
	}

	last := map[string]int64{}
	seen := map[string]bool{}
	for _, p := range procs {
		session, ok := servers[p.ppid]
		if !ok {
			continue
		}
		tty, ok := ptsPath(p.ttyNr)
		if !ok || seen[tty] {
			continue
		}
		seen[tty] = true
		var st syscall.Stat_t
		if syscall.Stat(tty, &st) != nil {
			continue
		}
		if at := st.Atim.Sec; at > last[session] {
			last[session] = at
		}
	}
	return last
}

type proc struct {
	pid, ppid, ttyNr int
	cmdline          []string
}

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
		procs = append(procs, proc{pid: pid, ppid: ppid, ttyNr: ttyNr, cmdline: cmdline})
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

// serverSession recognizes a `zellij --server SOCKET` process; the socket's
// file name is the session's.
func serverSession(cmdline []string) (string, bool) {
	for i, arg := range cmdline {
		if arg == "--server" && i+1 < len(cmdline) && i > 0 &&
			filepath.Base(cmdline[0]) == "zellij" {
			return filepath.Base(cmdline[i+1]), true
		}
	}
	return "", false
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
