package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func decode(t *testing.T, s string) map[string]json.RawMessage {
	t.Helper()
	m := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func field(t *testing.T, m map[string]json.RawMessage, keys ...string) string {
	t.Helper()
	return stored(m, keys...)
}

var now = time.Date(2026, 10, 7, 12, 0, 0, 0, time.FixedZone("", -7*3600))

func TestStatus(t *testing.T) {
	cases := map[string]string{
		"SessionStart":     "waiting",
		"Notification":     "waiting",
		"UserPromptSubmit": "working",
		"PostToolUse":      "working",
		"Stop":             "done",
		"SessionEnd":       "",
	}
	for event, want := range cases {
		if got, ok := Status(event); !ok || got != want {
			t.Errorf("Status(%q) = %q, %v", event, got, ok)
		}
	}
	if _, ok := Status("PreCompact"); ok {
		t.Error("Status accepted an ignored event")
	}
}

func TestMergeFirstEvent(t *testing.T) {
	captured := false
	next, rename := Merge(map[string]json.RawMessage{}, HookInput{
		Event: "SessionStart", SessionID: "abc", CWD: "/w",
	}, "waiting", HookEnv{
		InZellij: true, Session: "s1", Pane: "7", Now: now,
		Capture: func() (string, string, bool) { captured = true; return "3", "claude", true },
	})
	if !captured {
		t.Error("SessionStart didn't capture the tab")
	}
	if got := field(t, next, "updated_at"); got != "2026-10-07T12:00:00-07:00" {
		t.Errorf("updated_at = %q", got)
	}
	if got := string(next["zellij"]); got != `{"last_name":"claude","original_tab_name":"claude","pane_id":"7","session":"s1","tab_id":3}` {
		t.Errorf("zellij = %s", got)
	}
	if _, ok := next["message"]; ok {
		t.Error("message set outside a Notification")
	}
	if want := (&Rename{TabID: "3", Name: "○ claude"}); !reflect.DeepEqual(rename, want) {
		t.Errorf("rename = %+v, want %+v", rename, want)
	}
}

func TestMergeKeepsUnknownFields(t *testing.T) {
	old := decode(t, `{"custom":{"a":1},"message":"old","status":"working"}`)
	next, _ := Merge(old, HookInput{Event: "PostToolUse", SessionID: "abc"}, "working", HookEnv{Now: now})
	if string(next["custom"]) != `{"a":1}` {
		t.Errorf("custom = %s", next["custom"])
	}
	if field(t, next, "message") != "old" {
		t.Error("message dropped")
	}
	if _, ok := next["zellij"]; ok {
		t.Error("zellij set outside zellij")
	}
}

func TestMergeReusesTab(t *testing.T) {
	old := decode(t, `{"status":"working","zellij":{"session":"s1","pane_id":"7","tab_id":3,
		"original_tab_name":"claude","last_name":"Title"}}`)
	env := HookEnv{
		InZellij: true, Session: "s1", Pane: "7", Now: now, Title: "Title",
		Capture: func() (string, string, bool) { t.Error("captured needlessly"); return "", "", false },
	}
	_, rename := Merge(old, HookInput{Event: "PostToolUse", SessionID: "abc"}, "working", env)
	if rename != nil {
		t.Errorf("renamed with nothing changed: %+v", rename)
	}

	// The same agent resumed in another pane points at another tab.
	env.Pane = "9"
	env.Capture = func() (string, string, bool) { return "5", "Title", true }
	next, rename := Merge(old, HookInput{Event: "PostToolUse", SessionID: "abc"}, "working", env)
	if field(t, next, "zellij", "tab_id") != "5" {
		t.Errorf("tab_id = %s", next["zellij"])
	}
	// The captured name is this hook's own rename, so the original stays.
	if got := field(t, next, "zellij", "original_tab_name"); got != "claude" {
		t.Errorf("original_tab_name = %q", got)
	}
	if rename != nil {
		t.Errorf("renamed with nothing changed: %+v", rename)
	}
}

func TestMergeTruncatesName(t *testing.T) {
	long := strings.Repeat("é", 60)
	_, rename := Merge(map[string]json.RawMessage{}, HookInput{Event: "Stop", SessionID: "abc"}, "done", HookEnv{
		InZellij: true, Now: now, Title: long,
		Capture: func() (string, string, bool) { return "1", "tab", true },
	})
	if rename == nil || rename.Name != "✓ "+strings.Repeat("é", 49)+"…" {
		t.Errorf("rename = %+v", rename)
	}
}

func TestFindTab(t *testing.T) {
	panes := []byte(`[{"id":7,"is_plugin":true,"tab_id":0,"tab_name":"plugin"},
		{"id":7,"is_plugin":false,"tab_id":3,"tab_name":"✓ claude stuff"}]`)
	id, name, ok := FindTab(panes, "7")
	if !ok || id != "3" || name != "claude stuff" {
		t.Errorf("FindTab = %q, %q, %v", id, name, ok)
	}
	if _, _, ok := FindTab(panes, "8"); ok {
		t.Error("FindTab found a missing pane")
	}
}

func TestSessionTitle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.jsonl")
	var b strings.Builder
	b.WriteString(`{"type":"ai-title","aiTitle":"Old"}` + "\n")
	b.WriteString(`{"type":"agent-name","agentName":"Named"}` + "\n")
	// Push the title lines across several read chunks.
	filler := `{"type":"assistant","text":"` + strings.Repeat("x", 1000) + `"}` + "\n"
	for range 200 {
		b.WriteString(filler)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := SessionTitle(path); got != "Named" {
		t.Errorf("SessionTitle = %q", got)
	}

	// A file without a trailing newline, whose only match is the first line.
	if err := os.WriteFile(path, []byte(`{"type":"ai-title","aiTitle":"Only"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := SessionTitle(path); got != "Only" {
		t.Errorf("SessionTitle = %q", got)
	}
	if got := SessionTitle(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Errorf("SessionTitle of a missing file = %q", got)
	}
}

func TestStripGlyph(t *testing.T) {
	for in, want := range map[string]string{
		"- claude": "claude",
		"● claude": "claude",
		"○ claude": "claude",
		"✓ claude": "claude",
		"-x":       "-x",
		"claude":   "claude",
	} {
		if got := StripGlyph(in); got != want {
			t.Errorf("StripGlyph(%q) = %q, want %q", in, got, want)
		}
	}
}
