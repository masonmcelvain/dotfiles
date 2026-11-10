package ship

import "testing"

func TestParse(t *testing.T) {
	cases := map[string]Target{
		"US-1234": {Kind: Issue, Key: "US-1234"},
		"us-1234": {Kind: Issue, Key: "US-1234"},
		"https://acme.atlassian.net/browse/US-1234":                                         {Kind: Issue, Key: "US-1234"},
		"https://acme.atlassian.net/jira/software/projects/US/boards/1?selectedIssue=US-12": {Kind: Issue, Key: "US-12"},
		"https://bitbucket.org/acme/storefront/pull-requests/42/diff":                       {Kind: PR, Repo: "acme/storefront", Num: 42},
		"42":  {Kind: PR, Num: 42},
		"#12": {Kind: PR, Num: 12},
		"  make the locale picker skip unpublished locales  ": {Kind: Task, Text: "make the locale picker skip unpublished locales"},
		"https://example.com/browse/US-1":                     {Kind: Task, Text: "https://example.com/browse/US-1"},
	}
	for in, want := range cases {
		if got := Parse(in); got != want {
			t.Errorf("Parse(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestRemoteRepo(t *testing.T) {
	cases := map[string]string{
		"git@ssh.bitbucket.org:acme/storefront.git":     "acme/storefront",
		"git@bitbucket.org:acme/storefront.git":         "acme/storefront",
		"https://me@bitbucket.org/acme/storefront.git":  "acme/storefront",
		"https://bitbucket.org/acme/storefront/":        "acme/storefront",
		"https://github.com/masonmcelvain/dotfiles.git": "",
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
	if got := Branch(Parse("US-7")); got != "US-7" {
		t.Errorf("Branch(issue) = %q", got)
	}
	cases := map[string]Target{
		"storefront-US-1234":               Parse("US-1234"),
		"storefront-pr42":                  {Kind: PR, Num: 42},
		"storefront-fix-the-locale-picker": task,
	}
	for want, tg := range cases {
		if got := Session("storefront", tg); got != want {
			t.Errorf("Session(%+v) = %q, want %q", tg, got, want)
		}
	}
}

func TestPRHead(t *testing.T) {
	same := `{"pull_request": {
		"source": {"branch": {"name": "US-1234"}, "repository": {"full_name": "acme/storefront"}},
		"destination": {"branch": {"name": "master"}, "repository": {"full_name": "acme/storefront"}}}}`
	if got, err := PRHead([]byte(same)); err != nil || got != "US-1234" {
		t.Errorf("PRHead(same repo) = %q, %v", got, err)
	}
	fork := `{"pull_request": {
		"source": {"branch": {"name": "x"}, "repository": {"full_name": "someone/storefront"}},
		"destination": {"branch": {"name": "master"}, "repository": {"full_name": "acme/storefront"}}}}`
	if _, err := PRHead([]byte(fork)); err == nil {
		t.Error("PRHead accepted a fork's PR")
	}
	if _, err := PRHead([]byte(`{"pull_request": {}}`)); err == nil {
		t.Error("PRHead accepted a PR with no branch")
	}
}
