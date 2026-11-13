package zellij

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseSessions(t *testing.T) {
	out := "hop [Created 4months 21days 23h 51m 25s ago] (EXITED - attach to resurrect)\n" +
		"mason [Created 1day 6h 27m 45s ago] (current)\n" +
		"\n" +
		"fresh\n"
	want := []Session{
		{Name: "hop", Exited: true, Created: 4*2592000 + 21*86400 + 23*3600 + 51*60 + 25},
		{Name: "mason", Created: 86400 + 6*3600 + 27*60 + 45},
		{Name: "fresh"},
	}
	if got := ParseSessions(out); !reflect.DeepEqual(got, want) {
		t.Errorf("ParseSessions = %+v, want %+v", got, want)
	}
}

func TestParseLayoutCWD(t *testing.T) {
	layout := "layout {\n    cwd \"/home/me/Code\"\n    tab name=\"x\" {\n        pane cwd=\"sub\"\n        cwd \"/elsewhere\"\n    }\n}\n"
	if got := ParseLayoutCWD(layout); got != "/home/me/Code" {
		t.Errorf("ParseLayoutCWD = %q", got)
	}
	if got := ParseLayoutCWD("layout {\n}\n"); got != "" {
		t.Errorf("ParseLayoutCWD without cwd = %q", got)
	}
}

func TestServerSession(t *testing.T) {
	name, ok := serverSession([]string{"/home/me/bin/zellij", "--server", "/run/user/1000/zellij/0.45.1/che"})
	if !ok || name != "che" {
		t.Errorf("serverSession = %q, %v", name, ok)
	}
	if _, ok := serverSession([]string{"zellij", "attach", "che"}); ok {
		t.Error("serverSession matched a client")
	}
	if _, ok := serverSession([]string{"grep", "--server", "x"}); ok {
		t.Error("serverSession matched a non-zellij process")
	}
}

func TestFindTabPane(t *testing.T) {
	data := []byte(`[
		{"id": 0, "is_plugin": false, "tab_name": "sh"},
		{"id": 0, "is_plugin": true, "tab_name": "claude"},
		{"id": 2, "is_plugin": false, "tab_name": "claude"},
		{"id": 3, "is_plugin": false, "tab_name": "claude"}
	]`)
	if id, ok := FindTabPane(data, "claude"); !ok || id != 2 {
		t.Errorf("FindTabPane = %d, %v; want 2, true", id, ok)
	}
	if _, ok := FindTabPane(data, "hx"); ok {
		t.Error("FindTabPane found a pane in a tab that isn't there")
	}
}

func TestParseMetadataName(t *testing.T) {
	cases := map[string]string{
		"name \"acme-US-1234\"\ntabs {\n": "acme-US-1234",
		"name \"a \\\"quoted\\\" one\"\n": `a "quoted" one`,
		"tabs {\n}\n":                     "",
	}
	for in, want := range cases {
		if got := ParseMetadataName(in); got != want {
			t.Errorf("ParseMetadataName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSessionDir(t *testing.T) {
	info := t.TempDir()
	for dir, name := range map[string]string{"9c58259642a8": "hashed", "legacy": "legacy"} {
		os.MkdirAll(filepath.Join(info, dir), 0o755)
		os.WriteFile(filepath.Join(info, dir, "session-metadata.kdl"), []byte("name \""+name+"\"\n"), 0o644)
	}
	if got := sessionDir(info, "hashed"); got != filepath.Join(info, "9c58259642a8") {
		t.Errorf("hashed: got %q", got)
	}
	if got := sessionDir(info, "legacy"); got != filepath.Join(info, "legacy") {
		t.Errorf("legacy: got %q", got)
	}
	if got := sessionDir(info, "missing"); got != "" {
		t.Errorf("missing: got %q", got)
	}
}
