// Package slots starts fresh zellij sessions in the Code/ slots on cominor:
// /home/mmcelvain/Code is slot 1, /home/mmcelvain-N/Code is slot N, and a
// session started in slot N is named sN and uses the sh-hx-claude layout.
//
// A slot's last use is when its git index was last written: checkouts,
// commits, staging and the prompt's own `git status` all touch it, and it's
// per-worktree, unlike the shared refs. Slots that a live session is rooted
// in are skipped, so a new session never lands on work in progress.
package slots

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/masonmcelvain/dotfiles/mm/internal/gitx"
	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// Pattern is the glob the slots match, overridable with MM_SLOTS.
func Pattern() string {
	if p := os.Getenv("MM_SLOTS"); p != "" {
		return p
	}
	return "/home/mmcelvain*/Code"
}

// All lists every slot directory.
func All() []string {
	matches, _ := filepath.Glob(Pattern())
	var dirs []string
	for _, m := range matches {
		if fi, err := os.Stat(m); err == nil && fi.IsDir() {
			dirs = append(dirs, m)
		}
	}
	return dirs
}

// Available reports whether this host has slots at all, which is what gates
// `mm new` and the sessions picker's n key.
func Available() bool {
	return len(All()) > 0
}

// Num is a slot's number: /home/mmcelvain/Code is 1, /home/mmcelvain-4/Code
// is 4.
func Num(slot string) string {
	parent := filepath.Base(filepath.Dir(slot))
	if i := strings.LastIndexByte(parent, '-'); i >= 0 && i+1 < len(parent) {
		n := parent[i+1:]
		if strings.Trim(n, "0123456789") == "" {
			return n
		}
	}
	return "1"
}

// Name is the session name for a slot.
func Name(slot string) string {
	return "s" + Num(slot)
}

func realpath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// busy holds the names and real cwds of every live session. A session zellij
// hasn't written to its cache yet (a few seconds old) has only its name,
// which still marks an sN's slot busy.
type busy struct {
	names, cwds map[string]bool
}

func liveSessions() busy {
	b := busy{names: map[string]bool{}, cwds: map[string]bool{}}
	info := zellij.InfoDir()
	for _, s := range zellij.ListSessions() {
		if s.Exited {
			continue
		}
		b.names[s.Name] = true
		if cwd := zellij.SessionCWD(info, s.Name); cwd != "" {
			b.cwds[realpath(cwd)] = true
		}
	}
	return b
}

type slot struct {
	dir   string
	stamp time.Time
}

// order sorts slots least recently used first.
func order(slots []slot) {
	slices.SortStableFunc(slots, func(a, b slot) int {
		return cmp.Compare(a.stamp.UnixNano(), b.stamp.UnixNano())
	})
}

// Free lists every slot that no live session is rooted in, least recently
// used first.
func Free() []string {
	b := liveSessions()
	var free []slot
	for _, dir := range All() {
		if b.cwds[realpath(dir)] || b.names[Name(dir)] {
			continue
		}
		free = append(free, slot{dir: dir, stamp: lastUse(dir)})
	}
	order(free)
	dirs := make([]string, len(free))
	for i, s := range free {
		dirs[i] = s.dir
	}
	return dirs
}

// lastUse is the slot's index mtime, through rev-parse because most slots are
// worktrees, falling back to the directory's own mtime.
func lastUse(dir string) time.Time {
	if gitdir, err := gitx.Git(dir, "rev-parse", "--absolute-git-dir"); err == nil {
		if fi, err := os.Stat(filepath.Join(gitdir, "index")); err == nil {
			return fi.ModTime()
		}
	}
	if fi, err := os.Stat(dir); err == nil {
		return fi.ModTime()
	}
	return time.Time{}
}

var errNoSlot = errors.New("every Code slot has a live session in it")

// ensureBranch errors unless branch exists in repo, locally or on origin.
// Only a branch that isn't local is fetched, so the common case stays
// offline; fetching also refreshes a stale remote-tracking ref before the
// switch copies it.
func ensureBranch(repo, branch string) error {
	if _, err := gitx.Git(repo, "check-ref-format", "--branch", branch); err != nil {
		return fmt.Errorf("%s isn't a valid branch name", branch)
	}
	if _, err := gitx.Git(repo, "show-ref", "-q", "--verify", "refs/heads/"+branch); err == nil {
		return nil
	}
	refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)
	if _, err := gitx.Git(repo, "fetch", "-q", "origin", refspec); err != nil {
		return fmt.Errorf("no branch %s, here or on origin", branch)
	}
	return nil
}

// branchSlot picks a free slot with branch checked out, switching one over if
// none has it. A dirty slot is passed over rather than switched, since the
// switch could carry its changes onto the wrong branch or refuse outright.
func branchSlot(branch string) (string, error) {
	free := Free()
	if len(free) == 0 {
		return "", errNoSlot
	}
	if err := ensureBranch(free[0], branch); err != nil {
		return "", err
	}

	// git refuses to check a branch out twice, so a branch that's checked
	// out somewhere can only go there.
	if held, ok := gitx.Worktrees(free[0])[branch]; ok {
		held = realpath(held)
		for _, s := range free {
			if realpath(s) == held {
				return s, nil
			}
		}
		return "", fmt.Errorf("%s is checked out in %s, which isn't a free Code slot", branch, held)
	}

	for _, s := range free {
		if gitx.Dirty(s, true) {
			continue
		}
		if err := switchTo(s, branch); err != nil {
			return "", fmt.Errorf("couldn't check out %s in %s: %v", branch, s, err)
		}
		return s, nil
	}
	return "", errors.New("every free Code slot has uncommitted changes")
}

// switchTo runs git switch in slot dir with args, then brings its submodules
// along. With submodule.recurse set, git switch checks submodules out itself
// but never fetches them, so it fails outright when the new commit points a
// submodule at a commit the slot's own clone of it hasn't seen yet;
// submodule update fetches what's missing.
func switchTo(dir string, args ...string) error {
	steps := [][]string{
		append([]string{"-c", "submodule.recurse=false", "switch", "-q"}, args...),
		{"submodule", "update", "-q", "--init", "--recursive"},
	}
	for _, step := range steps {
		out, err := exec.Command("git", append([]string{"-C", dir}, step...)...).CombinedOutput()
		if err != nil {
			msg, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
			return errors.New(msg)
		}
	}
	return nil
}

// Start creates a background session in the least recently used free slot,
// with branch checked out when it's set, and returns the session's name and
// slot.
func Start(branch string) (name, dir string, err error) {
	if branch != "" {
		if dir, err = branchSlot(branch); err != nil {
			return "", "", err
		}
	} else {
		free := Free()
		if len(free) == 0 {
			return "", "", errNoSlot
		}
		dir = free[0]
	}
	return start(dir)
}

// StartNew is Start for a branch that doesn't exist yet: it creates branch
// from base in the least recently used clean free slot.
func StartNew(branch, base string) (name, dir string, err error) {
	free := Free()
	if len(free) == 0 {
		return "", "", errNoSlot
	}
	for _, s := range free {
		if gitx.Dirty(s, true) {
			continue
		}
		if err := switchTo(s, "-c", branch, base); err != nil {
			return "", "", fmt.Errorf("couldn't create %s in %s: %v", branch, s, err)
		}
		return start(s)
	}
	return "", "", errors.New("every free Code slot has uncommitted changes")
}

// start creates slot dir's session. The slot has no live session, so an sN
// still listed is an exited one, which Create clears rather than resurrects.
func start(dir string) (name, _ string, err error) {
	name = Name(dir)
	if err := zellij.Create(name, dir); err != nil {
		return "", "", err
	}
	return name, dir, nil
}
