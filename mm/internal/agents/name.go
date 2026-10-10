package agents

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/masonmcelvain/dotfiles/mm/internal/gitx"
	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// NamerEnv marks the `claude -p` the namer runs, so the hook ignores it even
// if its hooks somehow load.
const NamerEnv = "MM_TAB_NAMER"

// namerPrompt explains the tab bar to Haiku. Claude Code's own ai-title only
// sees the first prompt, so "review this pr" becomes "PR review"; the namer
// adds the branch, the PR, the agent's own reply and the sibling tabs.
const namerPrompt = `You name zellij terminal tabs. The user runs several Claude Code agents at once, one per tab, and glances at the tab bar to find the right one. Given what an agent is working on, reply with its tab label and nothing else.

Rules:
- At most 24 characters, about three words; count them. No quotes, no trailing punctuation.
- Lead with what distinguishes this work: a PR or issue number (#N), the feature, component or file.
- Add a short verb (QA, review, fix) only when it adds meaning. Never a generic label like "PR review", "Fix bug" or "Code changes".
- Never include the repo name (given below); the session name already shows it.
- Make it easy to tell apart from the other tabs listed.

Good: "Review #64980 DHL split", "QA #1065 hunk wrap", "GNOME dark mode", "Picker scroll bug".`

// NeedsName reports whether a hook event should start the namer: at the end
// of a turn, once Claude Code has titled the session, and again whenever that
// title changes, which is its sign the topic moved. An explicit name always
// wins, so it is never replaced.
func NeedsName(old map[string]json.RawMessage, in HookInput, env HookEnv) bool {
	return in.Event == "Stop" && env.InZellij && env.Title != "" && !env.Explicit &&
		stored(old, "label_for") != env.Title
}

// StartNamer runs `mm agent-status name ID` in its own session, so the hook
// returns at once and Claude Code doesn't wait on Haiku.
func StartNamer(sessionID string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "agent-status", "name", sessionID)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if cmd.Start() == nil {
		_ = cmd.Process.Release()
	}
}

// RunNamer asks Haiku for a descriptive label for agent sessionID's tab,
// stores it in the agent's state and renames the tab.
func RunNamer(sessionID string) error {
	file := filepath.Join(StateDir(), sessionID+".json")
	old, err := readState(file)
	if err != nil {
		return err
	}
	transcript := stored(old, "transcript_path")
	if transcript == "" {
		return errors.New("no transcript recorded")
	}
	if _, explicit := SessionTitle(transcript); explicit {
		return nil
	}
	prompts, reply := Brief(transcript)
	if len(prompts) == 0 {
		return errors.New("no prompts in the transcript")
	}

	cwd := stored(old, "cwd")
	session := stored(old, "zellij", "session")
	tab := stored(old, "zellij", "tab_id")
	var siblings []string
	if out, err := zellij.Action(time.Second, session, "list-panes", "-j", "-t"); err == nil {
		siblings = TabNames(out, tab)
	}
	branch := gitx.Branch(cwd)
	ctx := NameContext{
		Repo:     repoName(cwd),
		Branch:   branch,
		PR:       pullRequest(cwd, branch),
		Title:    stored(old, "title"),
		Siblings: siblings,
		Prompts:  prompts,
		Reply:    reply,
	}
	out, err := askHaiku(ctx.String())
	if err != nil {
		return err
	}
	label, ok := CleanLabel(out)
	if !ok {
		return fmt.Errorf("unusable label %q", out)
	}

	// Re-read: the hook kept writing while Haiku thought, and SessionEnd may
	// have removed the file.
	cur, err := readState(file)
	if err != nil {
		return err
	}
	data, _ := json.Marshal(label)
	cur["label"] = data
	show := truncateName(label)
	if z, ok := cur["zellij"]; ok {
		var m map[string]any
		if json.Unmarshal(z, &m) == nil {
			m["last_name"] = show
			cur["zellij"], _ = json.Marshal(m)
		}
	}
	if err := writeState(StateDir(), sessionID, cur); err != nil {
		return err
	}
	if tab != "" {
		_, err = zellij.Action(time.Second, session, "rename-tab", "--tab-id", tab, "--", tabLabel(stored(cur, "status"), show))
	}
	return err
}

// NameContext is what Haiku sees about an agent.
type NameContext struct {
	Repo, Branch, PR, Title string
	Siblings, Prompts       []string
	Reply                   string
}

func (c NameContext) String() string {
	var b strings.Builder
	line := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
	}
	line("Repo", c.Repo)
	line("Branch", c.Branch)
	line("Pull request", c.PR)
	line("Generic title", c.Title)
	line("Other tabs", strings.Join(c.Siblings, ", "))
	b.WriteString("\nUser's prompts:\n")
	for _, p := range c.Prompts {
		fmt.Fprintf(&b, "- %s\n", p)
	}
	if c.Reply != "" {
		fmt.Fprintf(&b, "\nAgent's latest reply:\n%s\n", c.Reply)
	}
	return b.String()
}

var commandTag = regexp.MustCompile(`(?s)<command-name>(.*?)</command-name>.*?<command-args>(.*?)</command-args>`)

// Brief pulls what the user asked for out of a transcript: the first and last
// few prompts, and the start of the agent's latest reply, where it states
// what it found or did. Tool results, injected context and subagents' turns
// are skipped.
func Brief(path string) (prompts []string, reply string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, ""
	}
	defer func() { _ = f.Close() }()

	var all []string
	r := bufio.NewReaderSize(f, 64<<10)
	for {
		line, err := r.ReadBytes('\n')
		if bytes.Contains(line, []byte(`"type":"user"`)) || bytes.Contains(line, []byte(`"type":"assistant"`)) {
			if p, ok := userPrompt(line); ok {
				all = append(all, clip(p, 400))
			} else if t := assistantText(line); t != "" {
				reply = t
			}
		}
		if err != nil {
			break
		}
	}
	if len(all) > 6 {
		all = append(all[:3:3], all[len(all)-3:]...)
	}
	return all, clip(reply, 2000)
}

type entry struct {
	Type        string `json:"type"`
	IsMeta      bool   `json:"isMeta"`
	IsSidechain bool   `json:"isSidechain"`
	Message     struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

func userPrompt(line []byte) (string, bool) {
	var e entry
	if json.Unmarshal(line, &e) != nil || e.Type != "user" || e.IsMeta || e.IsSidechain {
		return "", false
	}
	var s string
	if json.Unmarshal(e.Message.Content, &s) != nil {
		return "", false // tool results
	}
	s = strings.TrimSpace(s)
	if m := commandTag.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1] + " " + m[2]), true
	}
	for _, skip := range []string{"<task-notification", "<local-command", "<system-reminder", "Caveat:"} {
		if strings.HasPrefix(s, skip) {
			return "", false
		}
	}
	return s, s != ""
}

func assistantText(line []byte) string {
	var e entry
	if json.Unmarshal(line, &e) != nil || e.Type != "assistant" || e.IsSidechain {
		return ""
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	_ = json.Unmarshal(e.Message.Content, &blocks)
	var texts []string
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			texts = append(texts, b.Text)
		}
	}
	return strings.TrimSpace(strings.Join(texts, "\n"))
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// TabNames lists the tabs in `zellij action list-panes -j -t` other than
// except, without their status glyphs.
func TabNames(listPanes []byte, except string) []string {
	var panes []pane
	if json.Unmarshal(listPanes, &panes) != nil {
		return nil
	}
	var names []string
	seen := map[int]bool{}
	for _, p := range panes {
		if seen[p.TabID] || fmt.Sprint(p.TabID) == except {
			continue
		}
		seen[p.TabID] = true
		if n := StripGlyph(p.TabName); n != "" {
			names = append(names, n)
		}
	}
	return names
}

// CleanLabel turns Haiku's reply into a tab label, or rejects it when it
// plainly isn't one.
func CleanLabel(out string) (string, bool) {
	out = strings.TrimSpace(out)
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		out = out[:i]
	}
	out = StripGlyph(strings.TrimSpace(out))
	out = strings.TrimRight(out, ". ")
	out = strings.Trim(out, "\"'`*")
	out = strings.TrimRight(strings.TrimSpace(out), ".")
	n := len([]rune(out))
	if n == 0 || n > 60 {
		return "", false
	}
	return clip(out, 30), true
}

func repoName(cwd string) string {
	if top, err := gitx.Git(cwd, "rev-parse", "--show-toplevel"); err == nil && top != "" {
		return filepath.Base(top)
	}
	if cwd == "" {
		return ""
	}
	return filepath.Base(cwd)
}

// pullRequest is the open PR for branch, as "#N title", or "".
func pullRequest(cwd, branch string) string {
	if branch == "" || slices.Contains([]string{"main", "master"}, branch) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", "--json", "number,title", "-q", `"#\(.number) \(.title)"`)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// askHaiku runs a bare `claude -p` on Haiku: no settings, hooks, tools, MCP
// servers or saved session, so the call can't show up as an agent itself.
// It goes through the claude CLI rather than the API to reuse its login.
func askHaiku(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", "-p",
		"--model", "haiku",
		"--system-prompt", namerPrompt,
		"--setting-sources", "",
		"--settings", `{"disableAllHooks":true}`,
		"--strict-mcp-config",
		"--tools", "",
		"--no-session-persistence",
	)
	cmd.Dir = os.TempDir()
	cmd.Env = append(os.Environ(), NamerEnv+"=1")
	cmd.Stdin = strings.NewReader(prompt)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func readState(file string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]json.RawMessage{}
	}
	return m, nil
}

// writeState replaces an agent's state file atomically.
func writeState(dir, sessionID string, state map[string]json.RawMessage) error {
	out, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+sessionID+".*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(append(out, '\n'))
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, sessionID+".json")); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}
