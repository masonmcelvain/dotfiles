// Package ship hands a task to a Claude Code agent running the ship skill,
// in a checkout of its own: a Jira issue, a Bitbucket pull request, or a task
// described in words. It prepares the branch and a zellij session for it,
// starts Claude in the session's claude tab, and leaves it running in the
// background.
//
// The work goes in a git worktree under .claude/worktrees/, where Claude Code
// puts its own.
package ship

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/masonmcelvain/dotfiles/mm/internal/gitx"
	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// Kind is what a target names.
type Kind int

const (
	// Task is work described in words, with no issue behind it.
	Task Kind = iota
	Issue
	PR
)

// Target is the thing to ship, as given on the command line.
type Target struct {
	Kind Kind
	// Key is the Jira issue key, for an Issue: US-1234.
	Key string
	// Repo is WORKSPACE/SLUG when the target is a PR URL, "" otherwise.
	Repo string
	// Num is the PR number, for a PR.
	Num int
	// Text is the task description, for a Task.
	Text string
}

var (
	keyRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]+-\d+$`)
	// A Jira issue link: /browse/KEY, or a board or search with the issue
	// open in its side panel (?selectedIssue=KEY).
	jiraRE = regexp.MustCompile(`^https://[^/]+\.atlassian\.net\S*?(?:/browse/|[?&]selectedIssue=)([A-Za-z][A-Za-z0-9]+-\d+)\b`)
	prRE   = regexp.MustCompile(`^https://bitbucket\.org/([^/]+/[^/]+)/pull-requests/(\d+)`)
)

// Parse reads a target: a Jira issue key or link, a Bitbucket PR link or
// number (with or without #), or anything else as a task description.
// Anything with a space in it is a task, so a link followed by more words
// never parses as the link alone.
func Parse(arg string) Target {
	arg = strings.TrimSpace(arg)
	if strings.ContainsAny(arg, " \t\n") {
		return Target{Kind: Task, Text: arg}
	}
	if keyRE.MatchString(arg) {
		return Target{Kind: Issue, Key: strings.ToUpper(arg)}
	}
	if m := jiraRE.FindStringSubmatch(arg); m != nil {
		return Target{Kind: Issue, Key: strings.ToUpper(m[1])}
	}
	if m := prRE.FindStringSubmatch(arg); m != nil {
		n, _ := strconv.Atoi(m[2])
		return Target{Kind: PR, Repo: m[1], Num: n}
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(arg, "#")); err == nil && n > 0 {
		return Target{Kind: PR, Num: n}
	}
	return Target{Kind: Task, Text: arg}
}

// SplitArgs splits mm ship's arguments into the target and a note for the
// agent. When the first argument is an issue or PR, the rest is the note;
// otherwise the whole line is a task in words, so it needn't be quoted.
func SplitArgs(args []string) (target, note string) {
	if len(args) > 1 && Parse(args[0]).Kind != Task {
		return args[0], strings.TrimSpace(strings.Join(args[1:], " "))
	}
	return strings.Join(args, " "), ""
}

var remoteRE = regexp.MustCompile(`bitbucket\.org[:/]([^/]+/[^/]+?)(\.git)?/?$`)

// RemoteRepo is the WORKSPACE/SLUG a Bitbucket remote URL points at, in
// either its https or ssh form, or "".
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

// Branch is the branch a new piece of work starts on: the issue key itself,
// or ship/<slug> for a task, which the skill renames once it has a key.
func Branch(t Target) string {
	if t.Kind == Task {
		return "ship/" + Slug(t.Text)
	}
	return t.Key
}

// Session names the zellij session for t in repo NAME:
// storefront-US-1234, storefront-pr42,
// storefront-fix-the-thing.
func Session(name string, t Target) string {
	switch t.Kind {
	case PR:
		return fmt.Sprintf("%s-pr%d", name, t.Num)
	case Task:
		return name + "-" + Slug(t.Text)
	default:
		return name + "-" + t.Key
	}
}

// Prompt is what Claude starts with. It names the skill rather than typing
// /ship, because Claude Code leaves a command-line prompt that starts with a
// slash in the input box instead of sending it. A note from me goes last,
// as my words.
func Prompt(arg string, t Target, branch, note string) string {
	what := arg
	switch t.Kind {
	case Task:
		what = fmt.Sprintf("this task: %q", t.Text)
	case Issue:
		what = "Jira issue " + t.Key
	case PR:
		what = fmt.Sprintf("Bitbucket PR %d", t.Num)
	}
	p := fmt.Sprintf("Use the ship skill on %s. mm ship already checked out %s here for it.", what, branch)
	if note != "" {
		p += " My note on it: " + note
	}
	return p
}

// Run is `mm ship TARGET`, run from inside a checkout of the target's repo.
// note is passed to the agent with the target. It returns a line to report.
func Run(arg, note string) (string, error) {
	cwd, _ := os.Getwd()
	top, err := gitx.Git(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("not in a git repo; cd into a checkout of the repo first")
	}
	remote, _ := gitx.Git(top, "remote", "get-url", "origin")
	repo := RemoteRepo(remote)
	if repo == "" {
		return "", errors.New("origin isn't a Bitbucket repo")
	}

	t := Parse(arg)
	if t.Kind == Task && Slug(t.Text) == "" {
		return "", errors.New("say what to ship: a Jira issue, a PR number or link, or a task in words")
	}
	if t.Repo != "" && !strings.EqualFold(t.Repo, repo) {
		return "", fmt.Errorf("%s is in %s, but this checkout is %s", arg, t.Repo, repo)
	}

	// A PR resumes on its own branch, and so does an issue whose branch
	// already exists, here or on origin. Anything else starts a new one.
	branch := Branch(t)
	existing := false
	if t.Kind == PR {
		head, err := prHead(top, t.Num)
		if err != nil {
			return "", err
		}
		branch, existing = head, true
	}
	if !existing {
		if _, err := gitx.Git(top, "show-ref", "-q", "--verify", "refs/heads/"+branch); err == nil {
			existing = true
		} else if _, err := gitx.Git(top, "ls-remote", "--exit-code", "--heads", "origin", branch); err == nil {
			existing = true
		}
	}

	base, err := defaultBranch(top, repo)
	if err != nil {
		return "", err
	}
	if _, err := gitx.Git(top, "fetch", "-q", "origin", base); err != nil {
		return "", fmt.Errorf("couldn't fetch origin/%s", base)
	}

	dir, err := worktree(top, branch, "origin/"+base, existing)
	if err != nil {
		return "", err
	}
	_, name, _ := strings.Cut(repo, "/")
	session := Session(name, t)
	if err := zellij.Create(session, dir); err != nil {
		return "", err
	}

	// Through an interactive shell, so the tab has a shell once Claude
	// exits and Claude gets the PATH that mise and fnm set up in .bashrc.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	argv := []string{shell, "-ic", `claude --permission-mode auto "$1"; exec "$0" -i`,
		shell, Prompt(arg, t, branch, note)}
	if err := zellij.RunInTab(session, "claude", dir, argv); err != nil {
		return "", fmt.Errorf("created %s but couldn't start Claude in it: %v", session, err)
	}
	return fmt.Sprintf("shipping %s in %s (%s, on %s)", arg, session, dir, branch), nil
}

// prView is the part of `bkt pr view --json` prHead reads.
type prView struct {
	PullRequest struct {
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
		} `json:"source"`
		Destination struct {
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
		} `json:"destination"`
	} `json:"pull_request"`
}

// PRHead reads PR's source branch out of `bkt pr view --json`, refusing a
// PR from a fork, whose branch can't be pushed to.
func PRHead(data []byte) (string, error) {
	var v prView
	if err := json.Unmarshal(data, &v); err != nil {
		return "", err
	}
	pr := v.PullRequest
	if pr.Source.Branch.Name == "" {
		return "", errors.New("bkt didn't say the PR's branch")
	}
	src, dst := pr.Source.Repository.FullName, pr.Destination.Repository.FullName
	if src != "" && dst != "" && !strings.EqualFold(src, dst) {
		return "", fmt.Errorf("the PR is from the fork %s, which ship can't push to", src)
	}
	return pr.Source.Branch.Name, nil
}

// prHead is the source branch of PR num in the repo checked out at dir.
func prHead(dir string, num int) (string, error) {
	cmd := exec.Command("bkt", "pr", "view", strconv.Itoa(num), "--json")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("couldn't read PR %d with bkt (is it logged in? `bkt auth status`)", num)
	}
	head, err := PRHead(out)
	if err != nil {
		return "", fmt.Errorf("PR %d: %v", num, err)
	}
	return head, nil
}

// defaultBranch is origin's default branch, from origin/HEAD when the clone
// recorded it and from Bitbucket when it didn't.
func defaultBranch(dir, repo string) (string, error) {
	if ref, err := gitx.Git(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(ref, "origin/"), nil
	}
	cmd := exec.Command("bkt", "api", "/repositories/"+repo, "--json", "--jq", ".mainbranch.name")
	cmd.Dir = dir
	out, err := cmd.Output()
	if b := strings.Trim(strings.TrimSpace(string(out)), `"`); err == nil && b != "" && b != "null" {
		return b, nil
	}
	return "", errors.New("couldn't tell origin's default branch")
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
// fetched from origin when it isn't local, or as a new branch off base. A
// clean worktree already at that path but on another branch or a detached
// HEAD, as deleting its session in `mm sessions` leaves it, is switched to
// branch in place.
func worktree(top, branch, base string, existing bool) (string, error) {
	if held, ok := gitx.Worktrees(top)[branch]; ok {
		return held, nil
	}
	main := filepath.Dir(commonDir(top))
	dir := filepath.Join(main, ".claude", "worktrees", strings.ReplaceAll(branch, "/", "-"))
	reuse := false
	if _, err := os.Stat(dir); err == nil {
		if !isWorktree(top, dir) {
			return "", fmt.Errorf("%s exists but isn't a worktree of this repo", dir)
		}
		if gitx.Dirty(dir) {
			return "", fmt.Errorf("%s has uncommitted changes, so ship won't switch it to %s", dir, branch)
		}
		reuse = true
	}

	// The same three cases either way: `worktree add` makes the directory,
	// and `switch` moves the one that's there.
	add := []string{"-C", top, "worktree", "add", "-q"}
	sw := []string{"-C", dir, "switch", "-q"}
	switch {
	case !existing:
		add = append(add, "-b", branch, dir, base)
		sw = append(sw, "-c", branch, base)
	default:
		if _, err := gitx.Git(top, "show-ref", "-q", "--verify", "refs/heads/"+branch); err != nil {
			refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)
			if _, err := gitx.Git(top, "fetch", "-q", "origin", refspec); err != nil {
				return "", fmt.Errorf("couldn't fetch %s", branch)
			}
			add = append(add, "--track", "-b", branch, dir, "origin/"+branch)
			sw = append(sw, "--track", "-c", branch, "origin/"+branch)
		} else {
			add = append(add, dir, branch)
			sw = append(sw, branch)
		}
	}
	cmd, verb := add, "add a worktree"
	if reuse {
		cmd, verb = sw, "switch "+dir
	}
	out, err := exec.Command("git", cmd...).CombinedOutput()
	if err != nil {
		msg, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
		return "", fmt.Errorf("couldn't %s for %s: %s", verb, branch, msg)
	}
	return dir, nil
}

// isWorktree reports whether dir is the top of a worktree of the repo at top.
func isWorktree(top, dir string) bool {
	t, err := gitx.Git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return false
	}
	rt, err1 := filepath.EvalSymlinks(t)
	rd, err2 := filepath.EvalSymlinks(dir)
	return err1 == nil && err2 == nil && rt == rd && commonDir(dir) == commonDir(top)
}
