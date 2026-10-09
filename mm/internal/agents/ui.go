package agents

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/masonmcelvain/dotfiles/mm/internal/picker"
	"github.com/masonmcelvain/dotfiles/mm/internal/text"
	"github.com/masonmcelvain/dotfiles/mm/internal/zellij"
)

// Config is the agents picker's frame.
var Config = picker.Config{
	Title:       "agents",
	Help:        "1-9 jump · j/k move · enter open · d dismiss · q quit",
	Empty:       "no agents running",
	DetailLines: 2,
}

// Source lists agents for the picker and jumps to their zellij tabs,
// attaching or switching sessions as needed.
type Source struct {
	dir    string
	agents []Agent
	// sessions maps zellij session names to "live" or "exited"; a session
	// absent from it is gone.
	sessions map[string]string
	now      time.Time
}

// NewSource lists the agents recorded in dir.
func NewSource(dir string) *Source {
	return &Source{dir: dir}
}

// Load implements picker.Source.
func (s *Source) Load() int {
	s.now = time.Now()
	s.sessions = map[string]string{}
	for _, z := range zellij.ListSessions() {
		s.sessions[z.Name] = "live"
		if z.Exited {
			s.sessions[z.Name] = "exited"
		}
	}
	s.agents = Load(s.dir)
	return len(s.agents)
}

// sessionLabel is the session column for agent a, and whether that session
// is still live.
func (s *Source) sessionLabel(a Agent) (string, bool) {
	zs := a.Session()
	if zs == "" {
		return "-", false
	}
	switch s.sessions[zs] {
	case "live":
		return zs, true
	case "exited":
		return zs + " (exited)", false
	default:
		return zs + " (gone)", false
	}
}

func glyph(status string) string {
	switch status {
	case "waiting":
		return picker.Orange + "○" + picker.FG
	case "done":
		return picker.Green + "✓" + picker.FG
	default:
		return picker.Blue + "-" + picker.FG
	}
}

// Render implements picker.Source.
func (s *Source) Render(w int) []picker.Row {
	labels := make([]string, len(s.agents))
	sessW := 3
	for i, a := range s.agents {
		labels[i], _ = s.sessionLabel(a)
		sessW = max(sessW, text.Width(labels[i]))
	}
	sessW = min(sessW, 24)
	titleW := max(w-3-2-sessW-2-4-1, 8)

	rows := make([]picker.Row, len(s.agents))
	for i, a := range s.agents {
		_, live := s.sessionLabel(a)
		rows[i] = picker.Row{
			Text: glyph(a.Status) + "  " + text.Fit(a.Name, titleW) + "  " +
				text.Fit(labels[i], sessW) + "  " +
				text.PadLeft(text.Age(int64(s.now.Sub(a.Updated).Seconds())), 4),
			Dim: !live && a.Session() != "",
		}
	}
	return rows
}

// Detail implements picker.Source.
func (s *Source) Detail(i, w int) []string {
	a := s.agents[i]
	lines := []string{text.Fit(text.Tilde(a.CWD), w)}
	if a.Message != "" {
		lines = append(lines, picker.Dim+text.Fit(a.Message, w))
	}
	return lines
}

// Key implements picker.Source: d dismisses the selected agent.
func (s *Source) Key(key string, sel int) picker.Action {
	if key != "d" || sel < 0 {
		return picker.Action{}
	}
	if id := s.agents[sel].SessionID; id != "" {
		_ = os.Remove(filepath.Join(s.dir, id+".json"))
	}
	return picker.Action{Reload: true}
}

// Open implements picker.Source: it jumps to the agent's tab.
func (s *Source) Open(i int) picker.Action {
	a := s.agents[i]
	zs, tab, pane := a.Session(), a.Tab(), a.Pane()
	if zs == "" {
		return picker.Action{Flash: "not in zellij"}
	}
	// Attaching to a deleted session would create a fresh one with its name.
	if _, ok := s.sessions[zs]; !ok {
		return picker.Action{Flash: fmt.Sprintf("session %s is gone · d to dismiss", zs)}
	}

	current, inside := zellij.InCurrent()
	switch {
	case !inside:
		// Outside zellij: pre-focus the tab, then attach. On an exited
		// session the server isn't running yet, so keep retrying the focus
		// from a detached process once attach resurrects it.
		if tab != "" {
			if _, err := zellij.Action(2*time.Second, zs, "go-to-tab-by-id", tab); err != nil {
				focusLater(zs, tab)
			}
		}
		return picker.Action{Exec: zellij.AttachArgv(zs)}
	case current == zs:
		_, _ = zellij.Action(2*time.Second, "", "go-to-tab-by-id", tab)
		return picker.Action{Quit: true}
	default:
		// Detaches this client from the current session (leaving it running)
		// and drops it on the agent's pane in the target session.
		var args []string
		if pane != "" {
			args = []string{"--pane-id", "terminal_" + pane}
		}
		if zellij.Switch(zs, args...) != nil {
			return picker.Action{Flash: "couldn't switch to " + zs}
		}
		return picker.Action{Quit: true}
	}
}

// focusLater focuses tab in session zs once its server is up, from a process
// that outlives this one, since this one is about to become `zellij attach`.
func focusLater(zs, tab string) {
	cmd := exec.Command("sh", "-c", `for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
	sleep 0.3
	zellij action go-to-tab-by-id "$1" >/dev/null 2>&1 && break
done`, "sh", tab)
	cmd.Env = append(os.Environ(), "ZELLIJ_SESSION_NAME="+zs)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if cmd.Start() == nil {
		_ = cmd.Process.Release()
	}
}

// PrintTable prints the agents as a plain table, for when stdout isn't a
// terminal.
func (s *Source) PrintTable(w io.Writer) {
	s.Load()
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, a := range s.agents {
		label, _ := s.sessionLabel(a)
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", a.Status, a.Name, label,
			text.Tilde(a.CWD), text.Age(int64(s.now.Sub(a.Updated).Seconds())))
	}
	_ = tw.Flush()
}
