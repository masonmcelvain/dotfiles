package sessions

import (
	"fmt"

	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// GoToBranch does what the picker's b key does without the picker: it goes
// to the session that has branch checked out, preferring a live one. It
// returns a command to exec when landing there means attaching.
func GoToBranch(branch string) (argv []string, msg string, err error) {
	name, ok := FindBranch(Collect(), branch)
	if !ok {
		return nil, "", fmt.Errorf("no session has %s checked out", branch)
	}
	if current, _ := zellij.InCurrent(); name == current {
		return nil, fmt.Sprintf("already in %s, on %s", name, branch), nil
	}
	argv, err = SwitchTo(name)
	return argv, "", err
}
