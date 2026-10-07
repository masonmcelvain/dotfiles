package sessions

import (
	"fmt"
	"strings"

	"github.com/masonmcelvain/dotfiles/mm/internal/picker"
	"github.com/masonmcelvain/dotfiles/mm/internal/slots"
	"github.com/masonmcelvain/dotfiles/mm/internal/text"
	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// Source is the sessions picker. It attaches from a plain shell and switches
// from inside a session, so it works either way.
type Source struct {
	rows   []Row
	hasNew bool
}

// NewSource builds the picker, offering n only where there are Code slots
// for `mm new` to use.
func NewSource() (*Source, picker.Config) {
	s := &Source{hasNew: slots.Available()}
	help := "1-9 jump · j/k move · enter open · b branch"
	if s.hasNew {
		help += " · n new"
	}
	help += " · d delete · q quit"
	return s, picker.Config{
		Title:       "sessions",
		Help:        help,
		Empty:       "no other sessions",
		DetailLines: 2,
	}
}

// Load implements picker.Source.
func (s *Source) Load() int {
	s.rows = Collect()
	return len(s.rows)
}

// Render implements picker.Source.
func (s *Source) Render(w int) []picker.Row {
	// Start every column at the width its longest value wants.
	nameW, branchW, pathW := 4, 6, 8
	for _, r := range s.rows {
		nameW = max(nameW, text.Width(r.Name))
		branchW = max(branchW, text.Width(r.Branch))
		pathW = max(pathW, text.Width(text.Tilde(r.CWD)))
	}
	// Leave room for the gaps, the age column and a spare column at the edge.
	nameW, branchW, pathW = Widths(nameW, branchW, pathW, w-12)

	rows := make([]picker.Row, len(s.rows))
	for i, r := range s.rows {
		dim := ""
		if r.Exited {
			dim = picker.Dim
		}
		line := text.Fit(r.Name, nameW) + "  "
		line += picker.Green + dim + text.Fit(r.Branch, branchW) + picker.Reset + dim + "  "
		line += picker.Dim + text.Fit(text.Tilde(r.CWD), pathW) + picker.Reset + dim + "  "
		if r.Exited {
			line += picker.Dim + text.PadLeft(r.Age, 5)
		} else {
			line += picker.Orange + text.PadLeft(r.Age, 5)
		}
		rows[i] = picker.Row{Text: line, Dim: r.Exited}
	}
	return rows
}

// Detail implements picker.Source.
func (s *Source) Detail(i, w int) []string {
	r := s.rows[i]
	state := picker.Dim + "live · last typed in " + r.Age + " ago"
	if r.Exited {
		state = picker.Dim + "exited " + r.Age + " ago · enter resurrects it"
	}
	return []string{text.Fit(text.Tilde(r.CWD), w), state}
}

// Open implements picker.Source.
func (s *Source) Open(i int) picker.Action {
	return open(s.rows[i].Name)
}

func open(name string) picker.Action {
	argv, err := SwitchTo(name)
	if err != nil {
		return picker.Action{Flash: err.Error()}
	}
	return picker.Action{Quit: true, Exec: argv}
}

// Key implements picker.Source: b goes to a branch, d deletes a session, and
// n starts a fresh one.
func (s *Source) Key(key string, sel int) picker.Action {
	switch key {
	case "b":
		return picker.Action{Read: &picker.Prompt{Text: "branch: ", Then: s.branch}}
	case "d":
		if sel < 0 {
			return picker.Action{}
		}
		return s.confirmDelete(s.rows[sel])
	case "n":
		if s.hasNew {
			return s.start("", "")
		}
	}
	return picker.Action{}
}

func (s *Source) confirmDelete(r Row) picker.Action {
	q := fmt.Sprintf("delete %s? y yes · else no", r.Name)
	if !r.Exited {
		q = fmt.Sprintf("kill and delete %s? y yes · else no", r.Name)
	}
	rows := s.rows
	return picker.Action{Ask: &picker.Prompt{Text: q, Then: func(answer string) picker.Action {
		if answer != "y" {
			return picker.Action{}
		}
		return picker.Action{Flash: deleteSession(rows, r), Reload: true}
	}}}
}

func (s *Source) branch(answer string) picker.Action {
	branch := strings.TrimSpace(answer)
	if branch == "" {
		return picker.Action{}
	}
	if name, ok := FindBranch(s.rows, branch); ok {
		if current, _ := zellij.InCurrent(); name == current {
			return picker.Action{Flash: fmt.Sprintf("already in %s, on %s", name, branch)}
		}
		return open(name)
	}
	if !s.hasNew {
		return picker.Action{Flash: "no session has " + branch + " checked out"}
	}
	// Checking the branch can mean a fetch; say so rather than look hung.
	return s.start(branch, "starting a session on "+branch+"…")
}

// start runs `mm new` off the UI loop, flashing busy meanwhile.
func (s *Source) start(branch, busy string) picker.Action {
	if busy == "" {
		busy = "starting a session…"
	}
	return picker.Action{Busy: busy, Run: func() picker.Action {
		argv, _, err := New(branch)
		if err != nil {
			return picker.Action{Flash: err.Error()}
		}
		return picker.Action{Quit: true, Exec: argv}
	}}
}
