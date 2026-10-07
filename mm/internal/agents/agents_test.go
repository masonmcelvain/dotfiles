package agents

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"a.json": `{"session_id":"a","status":"working","updated_at":"2026-10-07T12:00:00-07:00","cwd":"/x/proj"}`,
		"b.json": `{"session_id":"b","status":"done","updated_at":"2026-10-07T11:00:00-07:00","title":"line\nbreak"}`,
		"c.json": `{"session_id":"c","status":"waiting","updated_at":"2026-10-07T10:00:00-07:00",
			"zellij":{"original_tab_name":"tab","tab_id":"4"}}`,
		"d.json": `{"session_id":"d","status":"done","updated_at":"2026-10-07T11:30:00-07:00"}`,
		"e.json": `not json`,
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var ids, names []string
	for _, a := range Load(dir) {
		ids = append(ids, a.SessionID)
		names = append(names, a.Name)
	}
	if want := []string{"c", "d", "b", "a"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("order = %v, want %v", ids, want)
	}
	if want := []string{"tab", "untitled", "line break", "proj"}; !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}
