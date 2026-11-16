package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var input = Input{
	Box:    "Slip Box",
	Reason: "found walking up from /vault",
	Kinds: []Kind{
		{Name: "claim", Description: "Someone asserts this."},
		{Name: "idea", Description: "Your own thought."},
		{Name: "question", Description: "Something you are curious about."},
	},
	Cards: []Card{
		{Key: "1", Title: "Bread", Kind: "hub", Level: 1},
		{Key: "2", Title: "Why does dough need time", Kind: "question", Level: 1},
		{Key: "2a", Title: "Time develops flavour", Kind: "claim", Level: 2},
	},
	AskParent: true,
}

// drive sends messages to the model and runs the commands they return, as
// the Bubble Tea runtime would, dropping any that block (a cursor blink).
func drive(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()
	queue := append([]tea.Msg(nil), msgs...)
	for steps := 0; len(queue) > 0; steps++ {
		if steps > 500 {
			t.Fatal("too many messages")
		}
		msg := queue[0]
		queue = queue[1:]
		next, cmd := m.Update(msg)
		m = next.(Model)
		queue = append(run(cmd), queue...)
	}
	return m
}

func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		switch msg := msg.(type) {
		case tea.BatchMsg:
			var out []tea.Msg
			for _, c := range msg {
				out = append(out, run(c)...)
			}
			return out
		case tea.QuitMsg, nil:
			return nil
		default:
			return []tea.Msg{msg}
		}
	case <-time.After(50 * time.Millisecond):
		return nil // a timer, such as the cursor blink
	}
}

func key(t tea.KeyType) tea.Msg   { return tea.KeyMsg{Type: t} }
func typed(s string) tea.Msg      { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
func enter() tea.Msg              { return key(tea.KeyEnter) }
func down() tea.Msg               { return key(tea.KeyDown) }
func esc() tea.Msg                { return key(tea.KeyEsc) }
func view(m Model) string         { return m.View() }
func has(s, sub string) bool      { return strings.Contains(s, sub) }
func result(m Model) Result       { return m.Result() }
func start(in Input) Model        { return NewModel(in) }
func size(w, h int) tea.Msg       { return tea.WindowSizeMsg{Width: w, Height: h} }
func ctrlC() tea.Msg              { return key(tea.KeyCtrlC) }
func slash() tea.Msg              { return typed("/") }
func withKind(k string) Input     { in := input; in.Kind = k; return in }
func withUnder(u string) Input    { in := input; in.Under = u; in.AskParent = false; return in }
func withTitle(k, t string) Input { in := withKind(k); in.Title = t; return in }

func TestFullFlow(t *testing.T) {
	m := drive(t, start(input), size(100, 30))
	v := view(m)
	if !has(v, "zk new") || !has(v, "Slip Box") || !has(v, "found walking up from /vault") {
		t.Errorf("the header names the box and why:\n%s", v)
	}
	if !has(v, "claim") || !has(v, "Someone asserts this.") || !has(v, "Your own thought.") {
		t.Errorf("the kinds show with their descriptions:\n%s", v)
	}

	m = drive(t, m, down(), enter()) // idea
	if !has(view(m), "Title for a new idea") || !has(view(m), "Your own thought.") {
		t.Fatalf("title step:\n%s", view(m))
	}
	m = drive(t, m, enter())
	if !has(view(m), "A card needs a title.") {
		t.Error("an empty title is refused")
	}
	m = drive(t, m, typed("  Patience   is  an ingredient "), enter())
	if !has(view(m), "No parent: a new root") || !has(view(m), "2a  Time develops flavour") {
		t.Fatalf("parent step:\n%s", view(m))
	}
	m = drive(t, m, down(), down(), down(), enter()) // 2a
	if !m.Done() {
		t.Fatal("the flow should be done")
	}
	if got := result(m); got != (Result{Kind: "idea", Title: "Patience is an ingredient", Under: "2a"}) {
		t.Errorf("result: %+v", got)
	}
}

func TestHeaderFitsTheWindow(t *testing.T) {
	in := input
	in.Reason = "found walking up from " + strings.Repeat("/a-long-folder", 10)
	m := drive(t, start(in), size(60, 20))
	first := strings.SplitN(view(m), "\n", 2)[0]
	if n := len([]rune(stripANSI(first))); n > 60 || !strings.HasSuffix(stripANSI(first), "...)") {
		t.Errorf("header is %d wide: %q", n, stripANSI(first))
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			esc = true
		case esc && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			esc = false
		case !esc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestNewRoot(t *testing.T) {
	m := drive(t, start(input), enter(), typed("A thought"), enter(), enter())
	if got := result(m); got != (Result{Kind: "claim", Title: "A thought"}) || !m.Done() {
		t.Errorf("result: %+v", got)
	}
}

func TestSearchForAParent(t *testing.T) {
	m := drive(t, start(input), enter(), typed("T"), enter())
	m = drive(t, m, slash(), typed("flavour"), enter(), enter())
	if got := result(m); got.Under != "2a" || !m.Done() {
		t.Errorf("search picked %+v\n%s", got, view(m))
	}
}

func TestBackAndCancel(t *testing.T) {
	// Esc steps back: from parent to title, from title to kind.
	m := drive(t, start(input), enter(), typed("T"), enter(), esc())
	if !has(view(m), "Title for a new claim") {
		t.Fatalf("esc from parent goes back to the title:\n%s", view(m))
	}
	m = drive(t, m, esc())
	if !has(view(m), "What kind of card?") {
		t.Fatalf("esc from title goes back to the kind:\n%s", view(m))
	}
	m = drive(t, m, down(), down(), enter())
	if !has(view(m), "Title for a new question") || !has(view(m), "T") {
		t.Errorf("a new kind keeps the title typed:\n%s", view(m))
	}

	m = drive(t, start(input), down(), ctrlC())
	if !result(m).Cancelled || !m.Done() {
		t.Error("ctrl+c cancels")
	}
	m = drive(t, start(input), esc())
	if !result(m).Cancelled {
		t.Error("esc on the first step cancels")
	}
}

func TestCommandLineSkipsSteps(t *testing.T) {
	m := start(withKind("question"))
	if !has(view(m), "Title for a new question") {
		t.Errorf("a kind on the command line starts at the title:\n%s", view(m))
	}
	m = drive(t, start(withUnder("2")), enter(), typed("Child"), enter())
	if got := result(m); got != (Result{Kind: "claim", Title: "Child", Under: "2"}) || !m.Done() {
		t.Errorf("--under skips the parent step: %+v", got)
	}
	if m := start(withTitle("idea", "Given")); !has(view(m), "No parent") {
		t.Errorf("kind and title given, parent asked:\n%s", view(m))
	}
}
