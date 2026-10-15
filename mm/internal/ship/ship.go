// Package ship hands a task to a Claude Code agent running the ship skill,
// in a checkout of its own: a GitHub issue or PR, or a task described in
// words. It prepares the branch and a zellij session for it, starts Claude
// in the session's claude tab, and leaves it running in the background.
//
// Where the repo has Code slots (iFixit on cominor), the work goes in the
// least recently used free slot, since a slot comes with a dev environment
// already set up. Any other repo gets a git worktree under
// .claude/worktrees/, where Claude Code puts its own.
package ship

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/masonmcelvain/dotfiles/mm/internal/gitx"
	"github.com/masonmcelvain/dotfiles/mm/internal/slots"
	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// Kind is what a target names.
type Kind int

const (
	// Task is work described in words, with no issue behind it.
	Task Kind = iota
	Issue
	PR
	// Number is a bare number, an issue or a PR until GitHub says which.
	Number
)

// Target is the thing to ship, as given on the command line.
type Target struct {
	Kind Kind
	// Repo is OWNER/NAME when the target is a URL, "" otherwise.
	Repo string
	Num  int
	// Text is the task description, for a Task.
	Text string
}

var urlRE = regexp.MustCompile(`^https://github\.com/([^/]+/[^/]+)/(issues|pull)/(\d+)`)

// Parse reads a target: an issue or PR URL, a number with or without #, or
// anything else as a task description.
func Parse(arg string) Target {
	arg = strings.TrimSpace(arg)
	if m := urlRE.FindStringSubmatch(arg); m != nil {
		n, _ := strconv.Atoi(m[3])
		kind := Issue
		if m[2] == "pull" {
			kind = PR
		}
		return Target{Kind: kind, Repo: m[1], Num: n}
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(arg, "#")); err == nil && n > 0 {
		return Target{Kind: Number, Num: n}
	}
	return Target{Kind: Task, Text: arg}
}

var remoteRE = regexp.MustCompile(`github\.com[:/]([^/]+/[^/]+?)(\.git)?/?$`)

// RemoteRepo is the OWNER/NAME a GitHub remote URL points at, in either its
// https or ssh form, or "".
func RemoteRepo(url string) string {
	if m := remoteRE.FindStringSubmatch(strings.TrimSpace(url)); m != nil {
		return m[1]
	}
	return ""
}

var nonWord = regexp.MustCompile(`[^a-z0-9]+`)

// Slug is the first few words of s, lowercased and hyphenated, for naming a
// branch after a task.
func Slug(s string) string {
	words := strings.Fields(nonWord.ReplaceAllString(strings.ToLower(s), " "))
	if len(words) > 4 {
		words = words[:4]
	}
	return strings.Join(words, "-")
}

// Branch is the branch a new piece of work starts on. It's provisional: the
// skill renames it after the change once it knows what the change is.
func Branch(t Target) string {
	if t.Kind == Task {
		return "ship/" + Slug(t.Text)
	}
	return "ship/" + strconv.Itoa(t.Num)
}

// Session names the zellij session for t in repo NAME: ifixit-62287,
// ifixit-pr62290, dotfiles-fix-the-thing.
func Session(name string, t Target) string {
	switch t.Kind {
	case PR:
		return fmt.Sprintf("%s-pr%d", name, t.Num)
	case Task:
		return name + "-" + Slug(t.Text)
	default:
		return fmt.Sprintf("%s-%d", name, t.Num)
	}
}

// Prompt is what Claude starts with. It names the skill rather than typing
// /ship, because Claude Code leaves a command-line prompt that starts with a
// slash in the input box instead of sending it.
func Prompt(arg string, t Target, branch string) string {
	what := arg
	if t.Kind == Task {
		what = fmt.Sprintf("this task: %q", t.Text)
	}
	return fmt.Sprintf("Use the ship skill on %s. mm ship already checked out %s here for it.", what, branch)
}

// Run is `mm ship TARGET`, run from inside a checkout of the target's repo.
// It returns a line to report.
func Run(arg string) (string, error) {
	cwd, _ := os.Getwd()
	top, err := gitx.Git(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("not in a git repo; cd into a checkout of the repo first")
	}
	remote, _ := gitx.Git(top, "remote", "get-url", "origin")
	repo := RemoteRepo(remote)
	if repo == "" {
		return "", errors.New("origin isn't a GitHub repo")
	}

	t := Parse(arg)
	if t.Kind == Task && Slug(t.Text) == "" {
		return "", errors.New("say what to ship: an issue or PR number or URL, or a task in words")
	}
	if t.Repo != "" && !strings.EqualFold(t.Repo, repo) {
		return "", fmt.Errorf("%s is in %s, but this checkout is %s", arg, t.Repo, repo)
	}

	// A PR resumes on its own branch; anything else starts a new one.
	branch := Branch(t)
	existing := false
	if t.Kind == PR || t.Kind == Number {
		head, isPR, err := prHead(top, t.Num)
		switch {
		case err != nil:
			return "", err
		case isPR:
			t.Kind, branch, existing = PR, head, true
		case t.Kind == PR:
			return "", fmt.Errorf("no PR %d here", t.Num)
		default:
			t.Kind = Issue
		}
	}
	if !existing {
		if _, err := gitx.Git(top, "show-ref", "-q", "--verify", "refs/heads/"+branch); err == nil {
			// Shipped before: pick the work back up.
			existing = true
		}
	}

	base, err := defaultBranch(top)
	if err != nil {
		return "", err
	}
	if _, err := gitx.Git(top, "fetch", "-q", "origin", base); err != nil {
		return "", fmt.Errorf("couldn't fetch origin/%s", base)
	}

	var session, dir string
	if hasSlots(top) {
		if existing {
			session, dir, err = slots.Start(branch)
		} else {
			session, dir, err = slots.StartNew(branch, "origin/"+base)
		}
	} else {
		dir, err = worktree(top, branch, "origin/"+base, existing)
		if err == nil {
			session = Session(filepath.Base(repo), t)
			err = zellij.Create(session, dir)
		}
	}
	if err != nil {
		return "", err
	}

	// Through an interactive shell, so the tab has a shell once Claude
	// exits and Claude gets the PATH that mise and fnm set up in .bashrc.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	argv := []string{shell, "-ic", `claude --permission-mode auto "$1"; exec "$0" -i`,
		shell, Prompt(arg, t, branch)}
	if err := zellij.RunInTab(session, "claude", dir, argv); err != nil {
		return "", fmt.Errorf("created %s but couldn't start Claude in it: %v", session, err)
	}
	return fmt.Sprintf("shipping %s in %s (%s, on %s)", arg, session, dir, branch), nil
}

// prHead is the head branch of PR num, if num is a PR. The PR has to live in
// this repo: a fork's branch can't be pushed to.
func prHead(dir string, num int) (head string, isPR bool, err error) {
	cmd := exec.Command("gh", "pr", "view", strconv.Itoa(num), "--json", "headRefName,isCrossRepository",
		"--jq", `if .isCrossRepository then "" else .headRefName end`)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// Issues share the numbering, so most failures just mean num is one.
		return "", false, nil
	}
	head = strings.TrimSpace(string(out))
	if head == "" {
		return "", true, fmt.Errorf("PR %d is from a fork, which ship can't push to", num)
	}
	return head, true, nil
}

// defaultBranch is origin's default branch, from origin/HEAD when the clone
// recorded it and from GitHub when it didn't.
func defaultBranch(dir string) (string, error) {
	if ref, err := gitx.Git(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(ref, "origin/"), nil
	}
	cmd := exec.Command("gh", "repo", "view", "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
	cmd.Dir = dir
	out, err := cmd.Output()
	if b := strings.TrimSpace(string(out)); err == nil && b != "" {
		return b, nil
	}
	return "", errors.New("couldn't tell origin's default branch")
}

// hasSlots reports whether dir is a checkout of the repo the Code slots are
// worktrees of.
func hasSlots(dir string) bool {
	all := slots.All()
	if len(all) == 0 {
		return false
	}
	return commonDir(dir) != "" && commonDir(dir) == commonDir(all[0])
}

func commonDir(dir string) string {
	d, err := gitx.Git(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(d); err == nil {
		return r
	}
	return d
}

// worktree returns the worktree for branch under the main checkout's
// .claude/worktrees/, adding it if it isn't there: on the existing branch,
// fetched from origin when it isn't local, or as a new branch off base.
func worktree(top, branch, base string, existing bool) (string, error) {
	if held, ok := gitx.Worktrees(top)[branch]; ok {
		return held, nil
	}
	main := filepath.Dir(commonDir(top))
	dir := filepath.Join(main, ".claude", "worktrees", strings.ReplaceAll(branch, "/", "-"))
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("%s exists but doesn't hold %s", dir, branch)
	}

	args := []string{"worktree", "add", "-q"}
	switch {
	case !existing:
		args = append(args, "-b", branch, dir, base)
	default:
		if _, err := gitx.Git(top, "show-ref", "-q", "--verify", "refs/heads/"+branch); err != nil {
			refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)
			if _, err := gitx.Git(top, "fetch", "-q", "origin", refspec); err != nil {
				return "", fmt.Errorf("couldn't fetch %s", branch)
			}
			args = append(args, "--track", "-b", branch, dir, "origin/"+branch)
		} else {
			args = append(args, dir, branch)
		}
	}
	out, err := exec.Command("git", append([]string{"-C", top}, args...)...).CombinedOutput()
	if err != nil {
		msg, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
		return "", fmt.Errorf("couldn't add a worktree for %s: %s", branch, msg)
	}
	return dir, nil
}
