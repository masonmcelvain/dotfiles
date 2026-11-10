package zellij

import (
	"slices"
	"testing"
)

func TestParsePS(t *testing.T) {
	out := `  301     1 ??       /opt/homebrew/bin/zellij --server /tmp/zellij-502/contract_version_1/che
  412   301 ttys003  -bash
  413   301 ??       /opt/homebrew/bin/zellij-plugin
  bad line
`
	got := parsePS(out)
	if len(got) != 3 {
		t.Fatalf("parsePS = %+v", got)
	}
	if got[0].pid != 301 || got[0].ppid != 1 || got[0].tty != "" ||
		!slices.Equal(got[0].cmdline, []string{"/opt/homebrew/bin/zellij", "--server", "/tmp/zellij-502/contract_version_1/che"}) {
		t.Errorf("server = %+v", got[0])
	}
	if got[1].tty != "/dev/ttys003" || got[1].ppid != 301 {
		t.Errorf("pane = %+v", got[1])
	}
}
