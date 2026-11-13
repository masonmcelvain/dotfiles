package sessions

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/masonmcelvain/dotfiles/mm/internal/gitx"
)

func TestWithin(t *testing.T) {
	cases := []struct {
		dir, root string
		want      bool
	}{
		{"/a/b", "/a/b", true},
		{"/a/b/c", "/a/b", true},
		{"/a/bc", "/a/b", false},
		{"/a", "/a/b", false},
		{"", "/a/b", false},
	}
	for _, c := range cases {
		if got := within(c.dir, c.root); got != c.want {
			t.Errorf("within(%q, %q) = %v, want %v", c.dir, c.root, got, c.want)
		}
	}
}

// repo makes a repo with one commit and a linked worktree on branch wt.
func repo(t *testing.T) (main, wt string) {
	t.Helper()
	t.Setenv("ZELLIJ_SESSION_NAME", "")
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	main, wt = filepath.Join(root, "main"), filepath.Join(root, "wt")
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run(root, "init", "-q", "-b", "main", main)
	run(main, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init")
	run(main, "worktree", "add", "-q", "-b", "wt", wt)
	return main, wt
}

func TestPlanCleanupRemovesWorktree(t *testing.T) {
	main, wt := repo(t)
	if !gitx.LinkedWorktree(wt) || gitx.LinkedWorktree(main) {
		t.Fatal("LinkedWorktree misreads main or wt")
	}

	r := Row{Name: "s", Branch: "wt", CWD: wt}
	plan := planCleanup([]Row{r}, r)
	if plan.worktree != wt {
		t.Fatalf("plan = %+v, want worktree %s", plan, wt)
	}
	plan.run()
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree still there: %v", err)
	}
	if _, err := gitx.Git(main, "rev-parse", "--verify", "-q", "refs/heads/wt"); err != nil {
		t.Error("branch wt was deleted")
	}
}

func TestPlanCleanupKeeps(t *testing.T) {
	main, wt := repo(t)
	r := Row{Name: "s", Branch: "wt", CWD: wt}

	// Another live session inside the worktree.
	other := Row{Name: "o", CWD: filepath.Join(wt, "sub")}
	if plan := planCleanup([]Row{r, other}, r); plan != (cleanup{}) {
		t.Errorf("shared worktree: plan = %+v", plan)
	}

	// Uncommitted work.
	if err := os.WriteFile(filepath.Join(wt, "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if plan := planCleanup([]Row{r}, r); plan.worktree != "" || plan.note == "" {
		t.Errorf("dirty worktree: plan = %+v", plan)
	}

	// The main checkout only detaches.
	m := Row{Name: "m", Branch: "main", CWD: main}
	if plan := planCleanup([]Row{m}, m); plan.detach != main || plan.worktree != "" {
		t.Errorf("main checkout: plan = %+v", plan)
	}
}
