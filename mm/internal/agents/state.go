// Package agents tracks Claude Code agents: the hook that records each
// agent's state in ~/.cache/agents/<session_id>.json and mirrors it in its
// zellij tab's name, and the picker that lists them.
package agents

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// StateDir is where the hook keeps one JSON file per agent session.
func StateDir() string {
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		cache = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return filepath.Join(cache, "agents")
}

// State is the part of a state file the picker reads. The hook keeps every
// field, known or not, when it rewrites one.
type State struct {
	SessionID string      `json:"session_id"`
	Status    string      `json:"status"`
	UpdatedAt string      `json:"updated_at"`
	CWD       string      `json:"cwd"`
	Title     string      `json:"title"`
	Label     string      `json:"label"`
	Message   string      `json:"message"`
	Zellij    *ZellijInfo `json:"zellij"`
}

// ZellijInfo places an agent in zellij.
type ZellijInfo struct {
	Session string `json:"session"`
	PaneID  string `json:"pane_id"`
	// TabID is a number in the file, but read loosely so a hand-edited or
	// older file can't fail the whole list.
	TabID           json.Number `json:"tab_id"`
	OriginalTabName string      `json:"original_tab_name"`
	LastName        string      `json:"last_name"`
}

// Agent is a state file ready to show.
type Agent struct {
	State
	Updated time.Time
	// Name is the namer's label, else the title, else the tab's original
	// name, else the cwd's base name.
	Name string
}

// Load reads every state file in dir, ordered waiting first, then done, then
// working, and newest first within each.
func Load(dir string) []Agent {
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var agents []Agent
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var s State
		if json.Unmarshal(data, &s) != nil {
			continue
		}
		agents = append(agents, newAgent(s))
	}
	Sort(agents)
	return agents
}

func newAgent(s State) Agent {
	if s.Status == "" {
		s.Status = "working"
	}
	a := Agent{State: s}
	a.Updated, _ = time.Parse(time.RFC3339, s.UpdatedAt)
	a.Name = cmp.Or(s.Label, s.Title)
	if a.Name == "" && s.Zellij != nil {
		a.Name = s.Zellij.OriginalTabName
	}
	if a.Name == "" && s.CWD != "" {
		a.Name = filepath.Base(s.CWD)
	}
	if a.Name == "" {
		a.Name = "untitled"
	}
	a.Name = oneLine(a.Name)
	a.Message = oneLine(a.Message)
	return a
}

// Sort orders agents waiting first, then done, then working, and newest first
// within each.
func Sort(agents []Agent) {
	slices.SortStableFunc(agents, func(a, b Agent) int {
		return cmp.Or(
			cmp.Compare(rank(a.Status), rank(b.Status)),
			b.Updated.Compare(a.Updated),
		)
	})
}

func rank(status string) int {
	switch status {
	case "waiting":
		return 0
	case "done":
		return 1
	default:
		return 2
	}
}

// Session is the zellij session the agent runs in, or "".
func (a Agent) Session() string {
	if a.Zellij == nil {
		return ""
	}
	return a.Zellij.Session
}

// Tab is the agent's zellij tab id, or "".
func (a Agent) Tab() string {
	if a.Zellij == nil {
		return ""
	}
	return a.Zellij.TabID.String()
}

// Pane is the agent's zellij pane id, or "".
func (a Agent) Pane() string {
	if a.Zellij == nil {
		return ""
	}
	return a.Zellij.PaneID
}

func oneLine(s string) string {
	return strings.NewReplacer("\n", " ", "\t", " ").Replace(s)
}
