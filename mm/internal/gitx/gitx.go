// Package gitx reads branches straight out of .git, and runs the few git
// commands that need git itself.
package gitx

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Branch is the branch checked out in the repo containing dir, a short sha
// for a detached HEAD, or "" outside a repo. It reads .git/HEAD rather than
// running git, since a git per session is what would make the picker slow.
// Worktrees and submodules, where .git is a file pointing elsewhere, work too.
func Branch(dir string) string {
	if dir == "" {
		return ""
	}
	for !exists(filepath.Join(dir, ".git")) {
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}

	gitdir := filepath.Join(dir, ".git")
	if fi, err := os.Stat(gitdir); err == nil && !fi.IsDir() {
		data, err := os.ReadFile(gitdir)
		if err != nil {
			return ""
		}
		_, target, ok := strings.Cut(firstLine(string(data)), " ")
		if !ok {
			return ""
		}
		gitdir = target
		if !filepath.IsAbs(gitdir) {
			gitdir = filepath.Join(dir, gitdir)
		}
	}
	data, err := os.ReadFile(filepath.Join(gitdir, "HEAD"))
	if err != nil {
		return ""
	}
	return ParseHead(firstLine(string(data)))
}

// ParseHead turns the contents of HEAD into a branch name.
func ParseHead(head string) string {
	head = strings.TrimSpace(head)
	switch {
	case strings.HasPrefix(head, "ref: refs/heads/"):
		return strings.TrimPrefix(head, "ref: refs/heads/")
	case strings.HasPrefix(head, "ref: "):
		return strings.TrimPrefix(head, "ref: ")
	case len(head) > 7:
		return head[:7]
	default:
		return head
	}
}

// Git runs git in dir and returns its trimmed stdout.
func Git(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	return strings.TrimSpace(string(out)), err
}

// Dirty reports whether dir has uncommitted changes, counting untracked files
// unless trackedOnly. --no-optional-locks keeps status from refreshing, and
// so touching, the index.
func Dirty(dir string, trackedOnly bool) bool {
	args := []string{"--no-optional-locks", "-C", dir, "status", "--porcelain"}
	if trackedOnly {
		args = append(args, "-uno")
	}
	out, _ := exec.Command("git", args...).Output()
	return len(strings.TrimSpace(string(out))) > 0
}

// Worktrees maps each branch checked out in the repo at dir to the worktree
// holding it.
func Worktrees(dir string) map[string]string {
	out, err := Git(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil
	}
	return ParseWorktrees(out)
}

// ParseWorktrees reads `git worktree list --porcelain`.
func ParseWorktrees(out string) map[string]string {
	held := map[string]string{}
	var path string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			path = p
		} else if b, ok := strings.CutPrefix(line, "branch refs/heads/"); ok {
			held[b] = path
		}
	}
	return held
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
