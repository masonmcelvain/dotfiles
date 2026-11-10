package zellij

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// readProcs lists every process from ps, since macOS has no /proc.
func readProcs() []proc {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,tty=,command=").Output()
	if err != nil {
		return nil
	}
	return parsePS(string(out))
}

// parsePS reads `ps -axo pid=,ppid=,tty=,command=` lines. ps joins the
// argv with spaces, so an argument with a space in it splits in two; the
// zellij server's socket path, the one argument that matters, has none.
func parsePS(out string) []proc {
	var procs []proc
	for line := range strings.Lines(out) {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			continue
		}
		var tty string
		if f[2] != "??" {
			tty = "/dev/" + f[2]
		}
		procs = append(procs, proc{pid: pid, ppid: ppid, tty: tty, cmdline: f[3:]})
	}
	return procs
}

func ttyAtime(tty string) (int64, bool) {
	var st syscall.Stat_t
	if syscall.Stat(tty, &st) != nil {
		return 0, false
	}
	return st.Atimespec.Sec, true
}
