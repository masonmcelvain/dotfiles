package gitx

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBranch(t *testing.T) {
	root := t.TempDir()

	// A plain repo, read from a subdirectory.
	write(t, filepath.Join(root, "repo/.git/HEAD"), "ref: refs/heads/main\n")
	if err := os.MkdirAll(filepath.Join(root, "repo/a/b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Branch(filepath.Join(root, "repo/a/b")); got != "main" {
		t.Errorf("plain repo: %q", got)
	}

	// A worktree, whose .git file points at a relative gitdir.
	write(t, filepath.Join(root, "repo/.git/worktrees/wt/HEAD"), "ref: refs/heads/feature/x\n")
	write(t, filepath.Join(root, "wt/.git"), "gitdir: ../repo/.git/worktrees/wt\n")
	if got := Branch(filepath.Join(root, "wt")); got != "feature/x" {
		t.Errorf("worktree: %q", got)
	}

	// A detached HEAD.
	write(t, filepath.Join(root, "det/.git/HEAD"), "0123456789abcdef0123456789abcdef01234567\n")
	if got := Branch(filepath.Join(root, "det")); got != "0123456" {
		t.Errorf("detached: %q", got)
	}

	if got := Branch(""); got != "" {
		t.Errorf("empty dir: %q", got)
	}
}

func TestParseHead(t *testing.T) {
	cases := map[string]string{
		"ref: refs/heads/main":      "main",
		"ref: refs/remotes/o/x":     "refs/remotes/o/x",
		"abcdef0123456789":          "abcdef0",
		"":                          "",
		"ref: refs/heads/a/b/c\n\n": "a/b/c",
	}
	for in, want := range cases {
		if got := ParseHead(in); got != want {
			t.Errorf("ParseHead(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseWorktrees(t *testing.T) {
	out := "worktree /home/me/Code\nHEAD abc\nbranch refs/heads/main\n\n" +
		"worktree /home/me-2/Code\nHEAD def\ndetached\n\n" +
		"worktree /home/me-3/Code\nHEAD 123\nbranch refs/heads/fix/thing\n"
	want := map[string]string{"main": "/home/me/Code", "fix/thing": "/home/me-3/Code"}
	if got := ParseWorktrees(out); !reflect.DeepEqual(got, want) {
		t.Errorf("ParseWorktrees = %v, want %v", got, want)
	}
}
