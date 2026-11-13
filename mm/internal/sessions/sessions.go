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
	"strings"
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

// cleanup is what deleting a session does to its checkout afterwards.
type cleanup struct {
	// worktree is a linked worktree to remove, freeing its branch with it.
	worktree string
	// detach is a main checkout to detach HEAD in, so another worktree can
	// check out branch; git refuses a branch that any worktree holds.
	detach string
	branch string
	// note says why the checkout was left alone, if it was.
	note string
}

// planCleanup decides what to do with session r's checkout once r is gone. A
// linked worktree is removed, keeping its branch and commits. The main
// checkout only detaches HEAD, keeping its files. Either is skipped while
// another live session, or this one, is still in the checkout, and a dirty
// tree is left as is: removing or detaching would strand uncommitted work.
func planCleanup(rows []Row, r Row) cleanup {
	if r.Branch == "" {
		return cleanup{}
	}
	if fi, err := os.Stat(r.CWD); err != nil || !fi.IsDir() {
		return cleanup{}
	}
	top, err := gitx.Git(r.CWD, "rev-parse", "--show-toplevel")
	if err != nil {
		return cleanup{}
	}

	for _, o := range rows {
		if o.Name != r.Name && !o.Exited && within(o.CWD, top) {
			return cleanup{}
		}
	}
	if current, _ := zellij.InCurrent(); current != "" &&
		within(zellij.SessionCWD(zellij.InfoDir(), current), top) {
		return cleanup{}
	}

	linked := gitx.LinkedWorktree(top)
	if !linked {
		if _, err := gitx.Git(top, "symbolic-ref", "-q", "HEAD"); err != nil {
			return cleanup{}
		}
	}
	if gitx.Dirty(top) {
		if linked {
			return cleanup{note: "kept worktree (uncommitted changes)"}
		}
		return cleanup{note: r.Branch + " still checked out (uncommitted changes)"}
	}
	if linked {
		return cleanup{worktree: top}
	}
	return cleanup{detach: top, branch: r.Branch}
}

// run carries out the plan, returning a note for the flash.
func (c cleanup) run() string {
	switch {
	case c.worktree != "":
		// Run from the shared git dir, since the worktree is going away.
		common, err := gitx.Git(c.worktree, "rev-parse", "--path-format=absolute", "--git-common-dir")
		if err == nil {
			_, err = gitx.Git(common, "worktree", "remove", c.worktree)
		}
		if err != nil {
			return "couldn't remove worktree " + text.Tilde(c.worktree)
		}
		return "removed worktree " + text.Tilde(c.worktree)
	case c.detach != "":
		if _, err := gitx.Git(c.detach, "switch", "-q", "--detach"); err != nil {
			return "couldn't release " + c.branch
		}
	}
	return c.note
}

// within reports whether dir is root or inside it.
func within(dir, root string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(root, dir)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
}

// deleteSession deletes session r, killing it first if it's live, then cleans
// up its checkout as planned. It returns the message to flash.
func deleteSession(r Row, plan cleanup) string {
	if zellij.Delete(r.Name) != nil {
		return "couldn't delete " + r.Name
	}
	msg := "deleted " + r.Name
	if note := plan.run(); note != "" {
		msg += " · " + note
	}
	return msg
}
