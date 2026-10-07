// Package sessions is the picker over zellij sessions, listing each one's git
// branch and working directory alongside its name.
//
// Session cwds come from zellij's own on-disk session info rather than from
// `zellij action dump-layout`: dump-layout needs a running server per session
// and hangs on exited ones, while the cache covers live and exited alike
// without spawning anything. Branches are read straight out of .git/HEAD for
// the same reason.
package sessions

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"text/tabwriter"
	"time"

	"github.com/masonmcelvain/dotfiles/mm/internal/gitx"
	"github.com/masonmcelvain/dotfiles/mm/internal/text"
	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// Row is one session, other than the current one.
type Row struct {
	Name   string
	Exited bool
	Branch string
	CWD    string
	// Used is when the session was last used: a live one's last keystroke,
	// an exited one's last metadata write.
	Used time.Time
	Age  string
}

// Collect lists every session but the current one, live ones first, then
// exited ones, each most recently used first.
func Collect() []Row {
	now := time.Now()
	info := zellij.InfoDir()
	lastInput := zellij.LastInput()
	current, _ := zellij.InCurrent()

	var rows []Row
	for _, s := range zellij.ListSessions() {
		// Switching to the session you're already in is a no-op; skip it.
		if s.Name == current {
			continue
		}
		cwd := zellij.SessionCWD(info, s.Name)
		r := Row{Name: s.Name, Exited: s.Exited, Branch: gitx.Branch(cwd), CWD: cwd}
		created := now.Add(-time.Duration(s.Created) * time.Second)
		if s.Exited {
			if mtime, ok := zellij.SessionMtime(info, s.Name); ok {
				r.Used = mtime
			} else {
				r.Used = created
			}
		} else if at, ok := lastInput[s.Name]; ok {
			r.Used = time.Unix(at, 0)
		} else {
			// A session with no pty panes left has no input clock; its
			// creation time is the best stand-in.
			r.Used = created
		}
		r.Age = text.Age(int64(now.Sub(r.Used).Seconds()))
		rows = append(rows, r)
	}
	Sort(rows)
	return rows
}

// Sort orders live sessions before exited ones, each most recently used
// first.
func Sort(rows []Row) {
	slices.SortStableFunc(rows, func(a, b Row) int {
		if a.Exited != b.Exited {
			if a.Exited {
				return 1
			}
			return -1
		}
		return cmp.Compare(b.Used.UnixNano(), a.Used.UnixNano())
	})
}

// FindBranch is the session with branch checked out: the current session if
// it has, else the first match in rows, which Collect orders best first.
func FindBranch(rows []Row, branch string) (string, bool) {
	if current, _ := zellij.InCurrent(); current != "" &&
		gitx.Branch(zellij.SessionCWD(zellij.InfoDir(), current)) == branch {
		return current, true
	}
	for _, r := range rows {
		if r.Branch == branch {
			return r.Name, true
		}
	}
	return "", false
}

// Widths fits the name, branch and path columns into avail columns. Width
// is only clawed back when the natural widths don't fit, so a wide terminal
// truncates nothing. The path yields first, then the branch, then the name:
// the branch is the thing worth reading here, and a truncated name stops
// identifying the session at all.
func Widths(nameW, branchW, pathW, avail int) (int, int, int) {
	over := nameW + branchW + pathW - avail
	give := func(w *int, floor int) {
		if over <= 0 {
			return
		}
		g := min(*w-floor, over)
		if g > 0 {
			*w -= g
			over -= g
		}
	}
	give(&pathW, 12)
	give(&branchW, 12)
	give(&nameW, 8)
	// Narrower than every floor combined: the path absorbs the remainder.
	if over > 0 {
		pathW -= over
	}
	return nameW, branchW, max(pathW, 4)
}

// PrintTable prints the sessions as a plain table, for when stdout isn't a
// terminal.
func PrintTable(w io.Writer, rows []Row) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, r := range rows {
		branch, state := r.Branch, "live"
		if branch == "" {
			branch = "-"
		}
		if r.Exited {
			state = "exited"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.Name, branch, text.Tilde(r.CWD), r.Age, state)
	}
	_ = tw.Flush()
}

// SwitchTo lands this terminal in session name: outside zellij by returning
// the attach command to exec, inside by switching this client over.
func SwitchTo(name string) (argv []string, err error) {
	if _, inside := zellij.InCurrent(); !inside {
		return zellij.AttachArgv(name), nil
	}
	if zellij.Switch(name) != nil {
		return nil, fmt.Errorf("couldn't switch to %s", name)
	}
	return nil, nil
}

// releaseBranch detaches HEAD in a deleted session's checkout so its branch
// is free to check out in another worktree; git refuses a branch that any
// worktree holds. The commit and files stay put. It returns why it didn't,
// if it didn't.
//
// A dirty tree keeps its branch: detaching would strand the uncommitted work
// away from the branch it belongs to. The index mtime is put back afterwards
// because `mm new` reads it as the slot's last use, and an abandoned slot
// shouldn't look like the freshest one.
func releaseBranch(rows []Row, name, cwd, branch string) string {
	if branch == "" {
		return ""
	}
	if fi, err := os.Stat(cwd); err != nil || !fi.IsDir() {
		return ""
	}

	// Another live session in the same checkout is still using the branch.
	for _, r := range rows {
		if r.Name != name && !r.Exited && r.CWD == cwd {
			return ""
		}
	}
	if current, _ := zellij.InCurrent(); current != "" &&
		zellij.SessionCWD(zellij.InfoDir(), current) == cwd {
		return ""
	}

	if _, err := gitx.Git(cwd, "symbolic-ref", "-q", "HEAD"); err != nil {
		return ""
	}
	if gitx.Dirty(cwd, false) {
		return branch + " still checked out (uncommitted changes)"
	}

	gitdir, err := gitx.Git(cwd, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return ""
	}
	index := filepath.Join(gitdir, "index")
	fi, statErr := os.Stat(index)
	if _, err := gitx.Git(cwd, "switch", "-q", "--detach"); err != nil {
		return "couldn't release " + branch
	}
	if statErr == nil {
		_ = os.Chtimes(index, time.Time{}, fi.ModTime())
	}
	return ""
}

// deleteSession deletes session r, killing it first if it's live, then releases
// its branch. It returns the message to flash.
func deleteSession(rows []Row, r Row) string {
	if zellij.Delete(r.Name) != nil {
		return "couldn't delete " + r.Name
	}
	msg := "deleted " + r.Name
	if note := releaseBranch(rows, r.Name, r.CWD, r.Branch); note != "" {
		msg += " · " + note
	}
	return msg
}
