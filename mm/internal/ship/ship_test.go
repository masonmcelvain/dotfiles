package ship

import "testing"

func TestParse(t *testing.T) {
	cases := map[string]Target{
		"https://github.com/iFixit/ifixit/issues/62287":     {Kind: Issue, Repo: "iFixit/ifixit", Num: 62287},
		"https://github.com/iFixit/ifixit/pull/62290/files": {Kind: PR, Repo: "iFixit/ifixit", Num: 62290},
		"62287": {Kind: Number, Num: 62287},
		"#12":   {Kind: Number, Num: 12},
		"  make the locale picker skip unpublished locales  ": {Kind: Task, Text: "make the locale picker skip unpublished locales"},
		"https://example.com/iFixit/ifixit/issues/1":          {Kind: Task, Text: "https://example.com/iFixit/ifixit/issues/1"},
	}
	for in, want := range cases {
		if got := Parse(in); got != want {
			t.Errorf("Parse(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestRemoteRepo(t *testing.T) {
	cases := map[string]string{
		"git@github.com:iFixit/ifixit.git":                 "iFixit/ifixit",
		"https://github.com/masonmcelvain/dotfiles.git":    "masonmcelvain/dotfiles",
		"https://github.com/masonmcelvain/dotfiles":        "masonmcelvain/dotfiles",
		"ssh://git@github.com/masonmcelvain/dotfiles.git/": "masonmcelvain/dotfiles",
		"https://gitlab.com/x/y.git":                       "",
	}
	for in, want := range cases {
		if got := RemoteRepo(in); got != want {
			t.Errorf("RemoteRepo(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNames(t *testing.T) {
	task := Parse("Fix the Locale picker's unpublished routes!")
	if got := Branch(task); got != "ship/fix-the-locale-picker" {
		t.Errorf("Branch(task) = %q", got)
	}
	if got := Branch(Target{Kind: Issue, Num: 7}); got != "ship/7" {
		t.Errorf("Branch(issue) = %q", got)
	}
	cases := map[string]Target{
		"ifixit-62287":                 {Kind: Issue, Num: 62287},
		"ifixit-pr62290":               {Kind: PR, Num: 62290},
		"ifixit-fix-the-locale-picker": task,
	}
	for want, tg := range cases {
		if got := Session("ifixit", tg); got != want {
			t.Errorf("Session(%+v) = %q, want %q", tg, got, want)
		}
	}
}
