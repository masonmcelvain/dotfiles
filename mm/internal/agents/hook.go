package agents

import (
	"bytes"
	"cmp"
	"encoding/json"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// HookInput is the part of a Claude Code hook event the hook reads.
type HookInput struct {
	Event          string `json:"hook_event_name"`
	SessionID      string `json:"session_id"`
	CWD            string `json:"cwd"`
	Message        string `json:"message"`
	TranscriptPath string `json:"transcript_path"`
}

// HookEnv is the zellij context the hook runs in.
type HookEnv struct {
	InZellij bool
	Session  string
	Pane     string
	Now      time.Time
	// Capture finds the tab holding this agent's pane, returning its id and
	// its name minus any status glyph.
	Capture func() (tabID, name string, ok bool)
	// Title is the session title from the transcript, or "".
	Title string
	// Explicit is whether Title is a name the user gave (claude -n, session
	// rename) rather than Claude Code's generated one.
	Explicit bool
}

// Rename is a tab rename the hook should make.
type Rename struct {
	TabID string
	Name  string
}

// Status maps a hook event to the agent status it means. ok is false for
// events the hook ignores; SessionEnd means "" (remove the state).
func Status(event string) (status string, ok bool) {
	switch event {
	case "SessionStart", "Notification":
		return "waiting", true
	case "UserPromptSubmit", "PostToolUse":
		return "working", true
	case "Stop":
		return "done", true
	case "SessionEnd":
		return "", true
	}
	return "", false
}

// Glyph is the status mark the hook puts in front of a tab's name.
func Glyph(status string) string {
	switch status {
	case "working":
		return "-"
	case "waiting":
		return "○"
	case "done":
		return "✓"
	}
	return ""
}

// stored reads a string field out of a state file, at path keys.
func stored(old map[string]json.RawMessage, keys ...string) string {
	cur := old
	for i, k := range keys {
		raw, ok := cur[k]
		if !ok {
			return ""
		}
		if i == len(keys)-1 {
			if k == "tab_id" {
				return tabID(raw)
			}
			var s string
			_ = json.Unmarshal(raw, &s)
			return s
		}
		cur = nil
		if json.Unmarshal(raw, &cur) != nil {
			return ""
		}
	}
	return ""
}

// Merge applies one hook event to an agent's old state, returning the new
// state and the tab rename to make, if any. Every field of the old state the
// hook doesn't own is kept.
func Merge(old map[string]json.RawMessage, in HookInput, status string, env HookEnv) (map[string]json.RawMessage, *Rename) {
	prevStatus := stored(old, "status")
	tab := stored(old, "zellij", "tab_id")
	orig := stored(old, "zellij", "original_tab_name")
	lastName := stored(old, "zellij", "last_name")
	prevSession := stored(old, "zellij", "session")
	prevPane := stored(old, "zellij", "pane_id")

	// Capture the agent's tab at session start, re-capture on every prompt
	// (the user may have renamed the tab or moved the pane), and otherwise
	// reuse the stored tab id -- unless it was recorded for a different
	// zellij session or pane, as after `claude --resume` in a new window,
	// where it now points at some unrelated tab. Renames target the stable
	// tab id, so they land on the agent's tab even while another is focused.
	if env.InZellij {
		recapture := in.Event == "SessionStart" || in.Event == "UserPromptSubmit" ||
			tab == "" || prevSession != env.Session || prevPane != env.Pane
		if recapture && env.Capture != nil {
			if id, name, ok := env.Capture(); ok {
				tab = id
				// Keep the stored name when the capture is empty (a renamed
				// tab with no name of its own) or is just the session title
				// this hook set on a previous rename.
				if name != "" && name != lastName {
					orig = name
				}
			}
		}
	}

	// The user's own name beats the namer's label, which beats Claude Code's
	// generated title.
	label := stored(old, "label")
	if env.Explicit {
		label = ""
	}
	show := cmp.Or(label, env.Title, orig)
	show = truncateName(show)

	next := maps.Clone(old)
	if next == nil {
		next = map[string]json.RawMessage{}
	}
	set := func(k string, v any) {
		data, _ := json.Marshal(v)
		next[k] = data
	}
	set("schema", 1)
	set("agent", "claude")
	set("session_id", in.SessionID)
	set("status", status)
	set("last_event", in.Event)
	set("updated_at", env.Now.Format(time.RFC3339))
	if in.CWD != "" {
		set("cwd", in.CWD)
	}
	if in.TranscriptPath != "" {
		set("transcript_path", in.TranscriptPath)
	}
	if env.Explicit {
		delete(next, "label")
	}
	if in.Event == "Notification" && in.Message != "" {
		set("message", in.Message)
	}
	if env.Title != "" {
		set("title", env.Title)
	}
	if tab != "" {
		id, _ := strconv.Atoi(tab)
		set("zellij", map[string]any{
			"session":           env.Session,
			"pane_id":           env.Pane,
			"tab_id":            id,
			"original_tab_name": orig,
			"last_name":         show,
		})
	}

	// PostToolUse fires on every tool call; rename only when the glyph or the
	// name changes (the session title appears and updates mid-session).
	var rename *Rename
	if env.InZellij && tab != "" && (status != prevStatus || show != lastName) {
		rename = &Rename{TabID: tab, Name: tabLabel(status, show)}
	}
	return next, rename
}

// truncateName keeps a tab name to 50 characters.
func truncateName(name string) string {
	return clip(name, 50)
}

// tabLabel is a tab's full name: the status glyph, then the shown name.
func tabLabel(status, show string) string {
	label := Glyph(status)
	if show != "" {
		label += " " + show
	}
	return label
}

// RunHook handles one Claude Code hook event from stdin. It must be fast and
// never fail the agent, so every error is swallowed.
func RunHook(stdin io.Reader) {
	if os.Getenv(NamerEnv) != "" {
		return
	}
	var in HookInput
	data, _ := io.ReadAll(stdin)
	if json.Unmarshal(data, &in) != nil || in.Event == "" || in.SessionID == "" {
		return
	}
	status, ok := Status(in.Event)
	if !ok {
		return
	}

	dir := StateDir()
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	file := filepath.Join(dir, in.SessionID+".json")
	old := map[string]json.RawMessage{}
	if data, err := os.ReadFile(file); err == nil {
		_ = json.Unmarshal(data, &old)
	}
	if old == nil {
		old = map[string]json.RawMessage{}
	}

	session, inZellij := zellij.InCurrent()
	if in.Event == "SessionEnd" {
		tab, orig := stored(old, "zellij", "tab_id"), stored(old, "zellij", "original_tab_name")
		if inZellij && tab != "" && orig != "" {
			_, _ = zellij.Action(time.Second, "", "rename-tab", "--tab-id", tab, orig)
		}
		_ = os.Remove(file)
		return
	}

	pane := os.Getenv("ZELLIJ_PANE_ID")
	title, explicit := SessionTitle(in.TranscriptPath)
	env := HookEnv{
		InZellij: inZellij,
		Session:  session,
		Pane:     pane,
		Now:      time.Now(),
		Capture:  func() (string, string, bool) { return captureTab(pane) },
		Title:    title,
		Explicit: explicit,
	}
	next, rename := Merge(old, in, status, env)
	// Claim the title before starting the namer, so the next Stop doesn't
	// start another.
	name := NeedsName(old, in, env) && next["zellij"] != nil
	if name {
		next["label_for"], _ = json.Marshal(title)
	}
	if writeState(dir, in.SessionID, next) != nil {
		return
	}

	if rename != nil {
		_, _ = zellij.Action(time.Second, "", "rename-tab", "--tab-id", rename.TabID, rename.Name)
	}
	if name {
		StartNamer(in.SessionID)
	}
}

type pane struct {
	ID       int    `json:"id"`
	IsPlugin bool   `json:"is_plugin"`
	TabID    int    `json:"tab_id"`
	TabName  string `json:"tab_name"`
}

// captureTab finds the tab holding pane, from the pane rather than from the
// focused tab: hooks fire while the user is off in another tab, and
// `current-tab-info` would report theirs. Plugin panes share the numeric id
// space with terminals (plugin_0 and terminal_0 are both 0), so skip them.
func captureTab(paneID string) (string, string, bool) {
	paneID = strings.TrimPrefix(paneID, "terminal_")
	if paneID == "" {
		return "", "", false
	}
	out, err := zellij.Action(time.Second, "", "list-panes", "-j", "-t")
	if err != nil {
		return "", "", false
	}
	return FindTab(out, paneID)
}

// FindTab picks pane paneID's tab out of `zellij action list-panes -j -t`,
// stripping the status glyph a previous rename added. The rest is the tab's
// real name, even when it starts with "claude".
func FindTab(listPanes []byte, paneID string) (string, string, bool) {
	var panes []pane
	if json.Unmarshal(listPanes, &panes) != nil {
		return "", "", false
	}
	for _, p := range panes {
		if p.IsPlugin || strconv.Itoa(p.ID) != paneID {
			continue
		}
		return strconv.Itoa(p.TabID), StripGlyph(p.TabName), true
	}
	return "", "", false
}

// StripGlyph removes a leading status glyph, and the space after it. The
// hyphen needs its space, so a tab really named "-x" keeps it; "●" was the
// working glyph before "-", and still marks tabs renamed by an older mm.
func StripGlyph(name string) string {
	if rest, ok := strings.CutPrefix(name, "- "); ok {
		return rest
	}
	for _, g := range []string{"●", "○", "✓"} {
		if rest, ok := strings.CutPrefix(name, g); ok {
			return strings.TrimPrefix(rest, " ")
		}
	}
	return name
}

// SessionTitle is the latest session title Claude Code wrote to the
// transcript: an explicit name (claude -n, session rename) lands as
// "agent-name", the generated title as "ai-title", and the newest entry wins.
// It reads backwards from the end, so it stays fast on large transcripts.
// explicit is whether the title is the user's own name. These entry types are
// undocumented internals: if they vanish, this returns "" and the tab keeps
// its original name.
func SessionTitle(path string) (title string, explicit bool) {
	if path == "" {
		return "", false
	}
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()
	line := lastLineMatching(f, func(line []byte) bool {
		return bytes.Contains(line, []byte(`"type":"agent-name"`)) ||
			bytes.Contains(line, []byte(`"type":"ai-title"`))
	})
	var entry struct {
		AgentName string `json:"agentName"`
		AITitle   string `json:"aiTitle"`
	}
	if line == nil || json.Unmarshal(line, &entry) != nil {
		return "", false
	}
	if entry.AgentName != "" {
		return entry.AgentName, true
	}
	return entry.AITitle, false
}

// lastLineMatching scans f backwards a chunk at a time for the last line that
// match accepts.
func lastLineMatching(f *os.File, match func([]byte) bool) []byte {
	fi, err := f.Stat()
	if err != nil {
		return nil
	}
	const chunk = 64 << 10
	end := fi.Size()
	// carry is the partial first line of the chunk after this one.
	var carry []byte
	for end > 0 {
		start := max(end-chunk, 0)
		buf := make([]byte, end-start, end-start+int64(len(carry)))
		if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
			return nil
		}
		buf = append(buf, carry...)
		for {
			i := bytes.LastIndexByte(buf, '\n')
			if i < 0 {
				break
			}
			if line := buf[i+1:]; match(line) {
				return line
			}
			buf = buf[:i]
		}
		carry = buf
		end = start
	}
	if len(carry) > 0 && match(carry) {
		return carry
	}
	return nil
}

// tabID parses a stored tab id, which may be a number or a numeric string.
func tabID(raw json.RawMessage) string {
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if _, err := strconv.Atoi(s); err == nil {
			return s
		}
	}
	return ""
}
