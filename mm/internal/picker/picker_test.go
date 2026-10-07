package picker

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type fake struct {
	n      int
	opened []int
	keys   []string
	action Action
}

func (f *fake) Load() int { return f.n }

func (f *fake) Render(int) []Row {
	rows := make([]Row, f.n)
	for i := range rows {
		rows[i] = Row{Text: "row" + strconv.Itoa(i)}
	}
	return rows
}

func (f *fake) Detail(i, _ int) []string { return []string{"detail" + strconv.Itoa(i)} }

func (f *fake) Open(i int) Action {
	f.opened = append(f.opened, i)
	return f.action
}

func (f *fake) Key(key string, _ int) Action {
	f.keys = append(f.keys, key)
	return f.action
}

func newModel(n int) (*model, *fake) {
	f := &fake{n: n}
	m := &model{cfg: Config{Title: "t", Help: "help", Empty: "empty", DetailLines: 2}, src: f, host: "h"}
	m.reload()
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 12})
	return m, f
}

func press(m *model, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		default:
			r := []rune(k)[0]
			msg = tea.KeyPressMsg{Code: r, Text: k}
		}
		m.Update(msg)
	}
}

func TestMovementWraps(t *testing.T) {
	m, _ := newModel(3)
	press(m, "k")
	if m.sel != 2 {
		t.Errorf("k from the top: sel = %d", m.sel)
	}
	press(m, "j")
	if m.sel != 0 {
		t.Errorf("j from the bottom: sel = %d", m.sel)
	}
	press(m, "G")
	if m.sel != 2 {
		t.Errorf("G: sel = %d", m.sel)
	}
	press(m, "g")
	if m.sel != 0 {
		t.Errorf("g: sel = %d", m.sel)
	}
}

func TestOpen(t *testing.T) {
	m, f := newModel(3)
	press(m, "2", "9", "j", "enter")
	if want := []int{1, 1}; len(f.opened) != 2 || f.opened[0] != want[0] || f.opened[1] != want[1] {
		t.Errorf("opened = %v, want %v", f.opened, want)
	}
}

func TestAsk(t *testing.T) {
	m, f := newModel(2)
	var answer string
	f.action = Action{Ask: &Prompt{Text: "sure?", Then: func(a string) Action {
		answer = a
		return Action{Flash: "did it"}
	}}}
	press(m, "d")
	if !strings.HasSuffix(m.render(), Orange+"sure?"+Reset) {
		t.Errorf("question not shown:\n%s", m.render())
	}
	f.action = Action{}
	// The answer goes to the prompt, not to the source.
	press(m, "y")
	if answer != "y" || len(f.keys) != 1 {
		t.Errorf("answer = %q, keys = %v", answer, f.keys)
	}
	if m.flash != "did it" {
		t.Errorf("flash = %q", m.flash)
	}
	// The next key clears the flash.
	press(m, "j")
	if m.flash != "" {
		t.Errorf("flash after a key = %q", m.flash)
	}
}

func TestRead(t *testing.T) {
	m, f := newModel(2)
	var answer string
	f.action = Action{Read: &Prompt{Text: "branch: ", Then: func(a string) Action {
		answer = a
		return Action{}
	}}}
	press(m, "b")
	f.action = Action{}
	press(m, "f", "o", "o", "enter")
	if answer != "foo" {
		t.Errorf("answer = %q", answer)
	}
	if m.mode != normal {
		t.Errorf("mode = %v", m.mode)
	}
}

func TestReadEscBacksOut(t *testing.T) {
	m, f := newModel(2)
	answer := "unset"
	f.action = Action{Read: &Prompt{Text: "branch: ", Then: func(a string) Action {
		answer = a
		return Action{}
	}}}
	press(m, "b", "x", "esc")
	if answer != "" || m.mode != normal {
		t.Errorf("answer = %q, mode = %v", answer, m.mode)
	}
}

func TestRenderLayout(t *testing.T) {
	m, _ := newModel(3)
	lines := strings.Split(m.render(), "\n")
	// header, gap, 3 rows, gap, sep, 2 detail, sep, hint
	if len(lines) != 11 {
		t.Fatalf("%d lines:\n%s", len(lines), m.render())
	}
	if !strings.Contains(lines[2], "->") || !strings.Contains(lines[2], "row0") {
		t.Errorf("first row = %q", lines[2])
	}
	if !strings.Contains(lines[7], "detail0") {
		t.Errorf("detail = %q", lines[7])
	}

	empty, _ := newModel(0)
	if !strings.Contains(empty.render(), "empty") {
		t.Errorf("no empty text:\n%s", empty.render())
	}
}

func TestScroll(t *testing.T) {
	m, _ := newModel(20)
	// 12 lines leave 12-6-2 = 4 for the list.
	press(m, "G")
	out := m.render()
	if !strings.Contains(out, "row19") || strings.Contains(out, "row15") {
		t.Errorf("not scrolled to the bottom:\n%s", out)
	}
}
