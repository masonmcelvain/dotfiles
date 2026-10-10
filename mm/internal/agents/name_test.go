package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNeedsName(t *testing.T) {
	env := HookEnv{InZellij: true, Title: "PR review"}
	stop := HookInput{Event: "Stop"}
	if !NeedsName(map[string]json.RawMessage{}, stop, env) {
		t.Error("first titled Stop didn't name")
	}
	if NeedsName(decode(t, `{"label_for":"PR review"}`), stop, env) {
		t.Error("named the same title twice")
	}
	if !NeedsName(decode(t, `{"label_for":"Old topic"}`), stop, env) {
		t.Error("a new title didn't rename")
	}
	if NeedsName(map[string]json.RawMessage{}, HookInput{Event: "PostToolUse"}, env) {
		t.Error("named mid-turn")
	}
	for _, e := range []HookEnv{
		{InZellij: true},
		{InZellij: true, Title: "Mine", Explicit: true},
		{Title: "PR review"},
	} {
		if NeedsName(map[string]json.RawMessage{}, stop, e) {
			t.Errorf("named with %+v", e)
		}
	}
}

func TestMergePrefersLabel(t *testing.T) {
	old := decode(t, `{"status":"working","label":"Review #42 DHL","zellij":{"session":"s1","pane_id":"7",
		"tab_id":3,"original_tab_name":"claude","last_name":"PR review"}}`)
	env := HookEnv{InZellij: true, Session: "s1", Pane: "7", Now: now, Title: "PR review"}
	next, rename := Merge(old, HookInput{Event: "Stop", SessionID: "abc"}, "done", env)
	if want := (&Rename{TabID: "3", Name: "✓ Review #42 DHL"}); !reflect.DeepEqual(rename, want) {
		t.Errorf("rename = %+v, want %+v", rename, want)
	}
	if field(t, next, "label") != "Review #42 DHL" {
		t.Error("label dropped")
	}

	// A name the user gives beats the label, and retires it.
	env.Title, env.Explicit = "mine", true
	next, rename = Merge(old, HookInput{Event: "Stop", SessionID: "abc"}, "done", env)
	if rename == nil || rename.Name != "✓ mine" {
		t.Errorf("rename = %+v", rename)
	}
	if _, ok := next["label"]; ok {
		t.Error("label kept under an explicit name")
	}
}

func TestBrief(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.jsonl")
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"review this pr"}}`,
		`{"type":"user","isMeta":true,"message":{"content":"injected"}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","content":"out"}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Looking."},{"type":"tool_use"}]}}`,
		`{"type":"assistant","isSidechain":true,"message":{"content":[{"type":"text","text":"subagent"}]}}`,
		`{"type":"user","message":{"content":"<task-notification>done</task-notification>"}}`,
		`{"type":"user","message":{"content":"<command-name>/review-loop</command-name>\n<command-message>review-loop</command-message>\n<command-args>64980</command-args>"}}`,
		`{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"x"},{"type":"text","text":"PR 64980 splits DHL tracking."}]}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	prompts, reply := Brief(path)
	if want := []string{"review this pr", "/review-loop 64980"}; !reflect.DeepEqual(prompts, want) {
		t.Errorf("prompts = %q, want %q", prompts, want)
	}
	if reply != "PR 64980 splits DHL tracking." {
		t.Errorf("reply = %q", reply)
	}
}

func TestBriefKeepsEnds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.jsonl")
	var b strings.Builder
	for _, p := range []string{"1", "2", "3", "4", "5", "6", "7", "8"} {
		b.WriteString(`{"type":"user","message":{"content":"` + p + `"}}` + "\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if prompts, _ := Brief(path); !reflect.DeepEqual(prompts, []string{"1", "2", "3", "6", "7", "8"}) {
		t.Errorf("prompts = %q", prompts)
	}
}

func TestCleanLabel(t *testing.T) {
	for in, want := range map[string]string{
		"Review #42 DHL split\n":        "Review #42 DHL split",
		`"QA #1065 hunk wrap".`:         "QA #1065 hunk wrap",
		"✓ GNOME dark mode\nbecause...": "GNOME dark mode",
		strings.Repeat("a", 40):         strings.Repeat("a", 29) + "…",
	} {
		if got, ok := CleanLabel(in); !ok || got != want {
			t.Errorf("CleanLabel(%q) = %q, %v, want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "  \n", strings.Repeat("word ", 20)} {
		if got, ok := CleanLabel(in); ok {
			t.Errorf("CleanLabel(%q) accepted %q", in, got)
		}
	}
}

func TestTabNames(t *testing.T) {
	panes := []byte(`[{"id":1,"tab_id":0,"tab_name":"- Picker scroll"},{"id":2,"tab_id":0,"tab_name":"- Picker scroll"},
		{"id":3,"tab_id":1,"tab_name":"✓ me"},{"id":4,"tab_id":2,"tab_name":"shell"}]`)
	if got := TabNames(panes, "1"); !reflect.DeepEqual(got, []string{"Picker scroll", "shell"}) {
		t.Errorf("TabNames = %q", got)
	}
}
