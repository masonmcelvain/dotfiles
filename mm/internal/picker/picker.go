// Package picker is the full-screen list picker shared by `mm agents` and
// `mm sessions`. It draws a header, the rows, a detail pane about the selected
// row and a key hint, reloading the rows on every refresh tick.
//
// Keys: 1-9 open, j/k or arrows move (wrapping around the ends), g/G
// top/bottom, enter open, r refresh, ctrl+l redraw, q or esc quit. Anything
// else goes to the Source's Key.
package picker

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/masonmcelvain/dotfiles/mm/internal/text"
)

// ANSI attributes shared by the picker and the rows its sources render.
const (
	Reset  = "\x1b[0m"
	Dim    = "\x1b[2m"
	Undim  = "\x1b[22m"
	FG     = "\x1b[39m"
	Orange = "\x1b[38;5;208m"
	Green  = "\x1b[32m"
	Blue   = "\x1b[34m"
)

// Config is the fixed text around the rows.
type Config struct {
	Title string
	Help  string
	// Empty stands in for the rows when there are none.
	Empty string
	// DetailLines is the height of the detail pane.
	DetailLines int
}

// Row is one rendered row. Text may carry ANSI attributes; Dim dims the whole
// row, gutter included.
type Row struct {
	Text string
	Dim  bool
}

// Source supplies the rows and acts on them.
type Source interface {
	// Load reloads the rows and returns their count.
	Load() int
	// Render returns every row, each at most width columns.
	Render(width int) []Row
	// Detail returns up to Config.DetailLines lines about row i.
	Detail(i, width int) []string
	// Open acts on row i.
	Open(i int) Action
	// Key handles a key the picker doesn't bind, with sel selected (-1 when
	// there are no rows).
	Key(key string, sel int) Action
}

// Action is what a Source asks the picker to do next.
type Action struct {
	// Flash replaces the key hint until the next key.
	Flash string
	// Reload reloads the rows.
	Reload bool
	// Quit leaves the picker.
	Quit bool
	// Exec leaves the picker and replaces this process with the command.
	Exec []string
	// Ask shows a question in place of the hint and answers it with the next
	// key. The rows don't refresh while it waits, so they can't shift under
	// the question.
	Ask *Prompt
	// Read reads a line of text in place of the hint. An empty answer is the
	// way to back out, and esc gives one.
	Read *Prompt
	// Run does slow work off the UI loop, showing Busy meanwhile, and then
	// carries out the Action it returns.
	Run  func() Action
	Busy string
}

// Prompt asks for an answer and continues with it.
type Prompt struct {
	Text string
	Then func(answer string) Action
}

type mode int

const (
	normal mode = iota
	asking
	reading
	busy
)

type model struct {
	cfg  Config
	src  Source
	host string

	n, sel, scroll int
	width, height  int
	flash          string

	mode   mode
	prompt *Prompt
	input  textinput.Model

	exec []string
}

type tickMsg struct{}

type doneMsg struct{ action Action }

// Run shows the picker until it quits, and returns the command an Action
// asked to exec, if any.
func Run(cfg Config, src Source) ([]string, error) {
	host, _ := os.Hostname()
	if h := os.Getenv("HOSTNAME"); h != "" {
		host = h
	}
	m := &model{cfg: cfg, src: src, host: host}
	m.reload()
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return nil, err
	}
	return final.(*model).exec, nil
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m *model) Init() tea.Cmd {
	return tick()
}

func (m *model) reload() {
	m.n = m.src.Load()
	m.sel = min(m.sel, m.n-1)
	m.sel = max(m.sel, 0)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.mode == normal {
			m.reload()
		}
		return m, nil
	case tickMsg:
		if m.mode == normal {
			m.reload()
		}
		return m, tick()
	case doneMsg:
		m.mode = normal
		m.flash = ""
		return m, m.apply(msg.action)
	case tea.KeyPressMsg:
		return m, m.key(msg)
	}
	if m.mode == reading {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) key(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	switch m.mode {
	case busy:
		if key == "ctrl+c" {
			return tea.Quit
		}
		return nil
	case asking:
		m.mode = normal
		m.flash = ""
		answer := msg.Text
		if answer == "" {
			answer = key
		}
		return m.apply(m.prompt.Then(answer))
	case reading:
		switch key {
		case "enter", "esc", "ctrl+c":
			answer := ""
			if key == "enter" {
				answer = m.input.Value()
			}
			m.mode = normal
			m.flash = ""
			return m.apply(m.prompt.Then(answer))
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	}

	m.flash = ""
	switch key {
	case "q", "esc", "ctrl+c":
		return tea.Quit
	// Moving past either end wraps around to the other. Tab and shift+tab
	// are undocumented aliases for j and k.
	case "j", "down", "tab":
		if m.n > 0 {
			m.sel = (m.sel + 1) % m.n
		}
	case "k", "up", "shift+tab":
		if m.n > 0 {
			m.sel = (m.sel - 1 + m.n) % m.n
		}
	case "g":
		m.sel = 0
	case "G":
		m.sel = max(m.n-1, 0)
	case "enter":
		if m.n > 0 {
			return m.apply(m.src.Open(m.sel))
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		i, _ := strconv.Atoi(key)
		if i-1 < m.n {
			return m.apply(m.src.Open(i - 1))
		}
	case "r":
		m.reload()
	// Wipe the screen too, for when something has drawn over it.
	case "ctrl+l":
		m.reload()
		return tea.ClearScreen
	default:
		sel := m.sel
		if m.n == 0 {
			sel = -1
		}
		return m.apply(m.src.Key(key, sel))
	}
	return nil
}

func (m *model) apply(a Action) tea.Cmd {
	if a.Flash != "" {
		m.flash = a.Flash
	}
	if a.Reload {
		m.reload()
	}
	switch {
	case a.Exec != nil:
		m.exec = a.Exec
		return tea.Quit
	case a.Quit:
		return tea.Quit
	case a.Ask != nil:
		m.mode = asking
		m.prompt = a.Ask
		m.flash = a.Ask.Text
	case a.Read != nil:
		m.mode = reading
		m.prompt = a.Read
		m.input = textinput.New()
		m.input.Prompt = ""
		m.input.SetWidth(max(m.width-text.Width(a.Read.Text)-5, 1))
		return m.input.Focus()
	case a.Run != nil:
		m.mode = busy
		if a.Busy != "" {
			m.flash = a.Busy
		}
		run := a.Run
		return func() tea.Msg { return doneMsg{run()} }
	}
	return nil
}

func (m *model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m *model) render() string {
	cols, lines := m.width, m.height
	if cols == 0 || lines == 0 {
		return ""
	}
	var b strings.Builder

	head := fmt.Sprintf("  %s · %d", m.cfg.Title, m.n)
	pad := max(cols-text.Width(head)-text.Width(m.host)-2, 1)
	b.WriteString(Dim + head + strings.Repeat(" ", pad) + m.host + Reset + "\n\n")

	// Rows sit behind an 8-column "  -> 1  " gutter.
	var rows []Row
	if m.n > 0 {
		rows = m.src.Render(cols - 8)
	}

	// Everything but the list: the header and its gap, the gap after the
	// list, two separators, the detail lines and the hint.
	listH := max(lines-6-m.cfg.DetailLines, 1)
	if m.sel < m.scroll {
		m.scroll = m.sel
	}
	if m.sel >= m.scroll+listH {
		m.scroll = m.sel - listH + 1
	}
	for i := m.scroll; i < len(rows) && i < m.scroll+listH; i++ {
		marker := "  "
		if i == m.sel {
			marker = "->"
		}
		num := " "
		if i < 9 {
			num = strconv.Itoa(i + 1)
		}
		attrs, numseq := "", Dim+num+Undim
		if rows[i].Dim {
			attrs, numseq = Dim, num
		}
		b.WriteString(attrs + "  " + marker + " " + numseq + "  " + rows[i].Text + Reset + "\n")
	}
	if m.n == 0 {
		b.WriteString("  " + Dim + m.cfg.Empty + Reset + "\n")
	}

	sep := "  " + Dim + strings.Repeat("─", max(cols-2, 0)) + Reset + "\n"
	var detail []string
	if m.n > 0 {
		detail = m.src.Detail(m.sel, cols-4)
	}
	b.WriteString("\n" + sep)
	for i := range m.cfg.DetailLines {
		line := ""
		if i < len(detail) {
			line = detail[i]
		}
		b.WriteString("   " + line + Reset + "\n")
	}
	b.WriteString(sep)

	switch {
	case m.mode == reading:
		b.WriteString("   " + Orange + m.prompt.Text + Reset + m.input.View())
	case m.flash != "":
		b.WriteString("   " + Orange + m.flash + Reset)
	default:
		b.WriteString("   " + Dim + m.cfg.Help + Reset)
	}
	return b.String()
}
