package sessions

import (
	"fmt"

	"github.com/masonmcelvain/dotfiles/mm/internal/slots"
	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// GoToBranch does what the picker's b key does without the picker: it goes
// to the session that has branch checked out, preferring a live one, or else
// starts a session on it where there are Code slots. It returns a command to
// exec when landing there means attaching.
func GoToBranch(branch string) (argv []string, msg string, err error) {
	if name, ok := FindBranch(Collect(), branch); ok {
		if current, _ := zellij.InCurrent(); name == current {
			return nil, fmt.Sprintf("already in %s, on %s", name, branch), nil
		}
		argv, err := SwitchTo(name)
		return argv, "", err
	}
	if !slots.Available() {
		return nil, "", fmt.Errorf("no session has %s checked out", branch)
	}
	return New(branch)
}

// New is `mm new [BRANCH]`: it starts a fresh session in the least recently
// used Code slot and lands this terminal in it.
func New(branch string) (argv []string, msg string, err error) {
	name, dir, err := slots.Start(branch)
	if err != nil {
		return nil, "", err
	}
	msg = fmt.Sprintf("%s in %s", name, dir)
	if branch != "" {
		msg += " on " + branch
	}
	argv, err = SwitchTo(name)
	if err != nil {
		return nil, msg, fmt.Errorf("created %s but couldn't switch to it", name)
	}
	return argv, msg, nil
}
