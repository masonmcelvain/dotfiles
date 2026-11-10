package zellij

import "path/filepath"

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
		if !ok || p.tty == "" || seen[p.tty] {
			continue
		}
		seen[p.tty] = true
		if at, ok := ttyAtime(p.tty); ok && at > last[session] {
			last[session] = at
		}
	}
	return last
}

// proc is a process as LastInput needs it: tty is the path of its
// controlling terminal, "" when it has none.
type proc struct {
	pid, ppid int
	tty       string
	cmdline   []string
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
