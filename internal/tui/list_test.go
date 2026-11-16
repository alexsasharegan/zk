package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexsasharegan/zk/internal/tree"
)

func listCard(k, title, kind, status string) ListCard {
	return ListCard{
		Card:   Card{Key: k, Title: title, Kind: kind, Status: status},
		Row:    fmt.Sprintf("%-10s %-9s %-9s %s", k, status, kind, title),
		Branch: fmt.Sprintf("%s  %s  (%s, %s)", k, title, kind, status),
	}
}

// listEnv is a list input that records what was opened.
type listEnv struct {
	in      ListInput
	opened  []string
	openErr error
}

func newListEnv() *listEnv {
	e := &listEnv{}
	e.in = ListInput{
		Box: "Slip Box", Reason: "the default in ~/.config/zk/config.toml", Root: "Slip Box",
		Glyphs: tree.Unicode, Where: "Obsidian", Editor: "nvim",
		Cards: []ListCard{
			listCard("1", "Bread", "hub", "linked"),
			listCard("1a", "Why does dough need time", "question", "raw"),
			listCard("1a1", "Time develops flavour", "claim", "proposed"),
			listCard("1a2", "Cold slows the yeast", "claim", "raw"),
			listCard("1b", "Salt tightens gluten", "claim", "raw"),
			listCard("2", "Ovens", "idea", "raw"),
		},
		Open: func(c ListCard) error { e.opened = append(e.opened, "obsidian "+c.Key); return e.openErr },
		Edit: func(c ListCard) error { e.opened = append(e.opened, "nvim "+c.Key); return nil },
	}
	return e
}

// start builds the list with a fake in place of handing over the terminal.
func (e *listEnv) start() ListModel {
	m := NewListModel(e.in)
	m.exec = func(fn func() error, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg {
			e.opened = append(e.opened, "terminal handed over")
			return done(fn())
		}
	}
	return m
}

func driveList(t *testing.T, m ListModel, msgs ...tea.Msg) ListModel {
	t.Helper()
	queue := append([]tea.Msg(nil), msgs...)
	for steps := 0; len(queue) > 0; steps++ {
		if steps > 500 {
			t.Fatal("too many messages")
		}
		next, cmd := m.Update(queue[0])
		m = next.(ListModel)
		queue = append(run(cmd), queue[1:]...)
	}
	return m
}

// screen is the list as drawn, without styling.
// The cursor's line is padded to the window; the padding is trimmed.
func screen(m ListModel) []string {
	lines := strings.Split(stripANSI(m.View()), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return lines
}

// cursorLine is the line the cursor is on.
func cursorLine(t *testing.T, m ListModel) string {
	t.Helper()
	for _, l := range screen(m) {
		if strings.HasPrefix(l, "> ") {
			return strings.TrimPrefix(l, "> ")
		}
	}
	t.Fatalf("no cursor:\n%s", stripANSI(m.View()))
	return ""
}

// cards is the keys of the cards drawn, top to bottom.
func cards(m ListModel) string {
	var keys []string
	for _, l := range m.lines {
		keys = append(keys, m.in.Cards[l.card].Key)
	}
	return strings.Join(keys, " ")
}

func TestListRows(t *testing.T) {
	e := newListEnv()
	m := driveList(t, e.start(), size(100, 20))
	s := screen(m)
	if s[0] != "zk list  Slip Box  (the default in ~/.config/zk/config.toml)" || s[1] != "6 cards" {
		t.Errorf("the header names the box and why, then the count:\n%s", strings.Join(s, "\n"))
	}
	for i, c := range e.in.Cards {
		mark := "  "
		if i == 0 {
			mark = "> "
		}
		if s[i+2] != mark+c.Row {
			t.Errorf("row %d: %q, want %q", i, s[i+2], mark+c.Row)
		}
	}
	if len(s) != 20 || !strings.Contains(s[19], "/: filter") || !strings.Contains(s[19], "q: quit") {
		t.Errorf("the keys are on the last of %d lines: %q", len(s), s[len(s)-1])
	}

	m = driveList(t, m, down(), typed("j"))
	if got := cursorLine(t, m); got != e.in.Cards[2].Row {
		t.Errorf("down and j move: %q", got)
	}
	m = driveList(t, m, typed("k"), key(tea.KeyUp), key(tea.KeyUp))
	if got := cursorLine(t, m); got != e.in.Cards[0].Row {
		t.Errorf("up and k move, and stop at the top: %q", got)
	}
	m = driveList(t, m, typed("G"))
	if got := cursorLine(t, m); got != e.in.Cards[5].Row {
		t.Errorf("G goes to the end: %q", got)
	}
	m = driveList(t, m, down(), typed("g"))
	if got := cursorLine(t, m); got != e.in.Cards[0].Row {
		t.Errorf("g goes to the top: %q", got)
	}
	// Folding keys do nothing in the rows.
	if m = driveList(t, m, typed(" "), key(tea.KeyLeft), typed("-")); cards(m) != "1 1a 1a1 1a2 1b 2" {
		t.Errorf("rows do not fold: %s", cards(m))
	}
}

func TestListScrollsInASmallWindow(t *testing.T) {
	m := driveList(t, newListEnv().start(), size(40, 6)) // room for three cards
	s := screen(m)
	if len(s) != 6 || !strings.HasPrefix(s[4], "  1a1") {
		t.Fatalf("three cards fit:\n%s", strings.Join(s, "\n"))
	}
	for _, l := range s {
		if len([]rune(l)) > 40 {
			t.Errorf("a line is wider than the window: %q", l)
		}
	}
	m = driveList(t, m, key(tea.KeyPgDown), down())
	s = screen(m)
	if !strings.HasPrefix(s[2], "  1a1") || !strings.HasPrefix(s[4], "> 1b") {
		t.Errorf("the window follows the cursor:\n%s", strings.Join(s, "\n"))
	}
	m = driveList(t, m, key(tea.KeyEnd), key(tea.KeyPgUp), key(tea.KeyPgUp))
	if got := cursorLine(t, m); !strings.HasPrefix(got, "1 ") {
		t.Errorf("page up stops at the top: %q", got)
	}
}

func TestListFilter(t *testing.T) {
	e := newListEnv()
	m := driveList(t, e.start(), size(100, 20), slash(), typed("flavour"))
	if cards(m) != "1a1" {
		t.Errorf("by title: %s", cards(m))
	}
	if s := screen(m); !strings.HasPrefix(s[1], "/flavour") || !strings.Contains(s[1], "1 of 6 cards") || !strings.Contains(s[19], "esc: clear it") {
		t.Errorf("the filter line and its keys:\n%s", strings.Join(s, "\n"))
	}
	// q is typed into the filter, not a quit.
	m = driveList(t, m, typed("q"))
	if m.quit || cards(m) != "" || !strings.Contains(stripANSI(m.View()), "No card matches the filter.") {
		t.Errorf("q while filtering: quit %v, cards %q", m.quit, cards(m))
	}
	m = driveList(t, m, esc())
	if m.quit || cards(m) != "1 1a 1a1 1a2 1b 2" {
		t.Errorf("esc clears the filter being typed: quit %v, cards %q", m.quit, cards(m))
	}

	for term, want := range map[string]string{
		"1a2":      "1a2",               // key
		"question": "1a",                // kind
		"proposed": "1a1",               // status
		"claim":    "1a1 1a2 1b",        // in the box's order
		"":         "1 1a 1a1 1a2 1b 2", // nothing typed
	} {
		if got := cards(driveList(t, e.start(), slash(), typed(term))); got != want {
			t.Errorf("filter %q: %q, want %q", term, got, want)
		}
	}

	// Enter keeps the filter; the list has the keys again. Esc clears it,
	// and a second esc quits.
	m = driveList(t, e.start(), size(100, 20), slash(), typed("claim"), enter(), down())
	if got := cursorLine(t, m); got != e.in.Cards[3].Row {
		t.Errorf("moving in a filtered list: %q", got)
	}
	if s := screen(m); !strings.HasPrefix(s[1], "/claim  3 of 6 cards") || !strings.Contains(s[19], "esc: clear the filter") {
		t.Errorf("a kept filter shows:\n%s", strings.Join(s, "\n"))
	}
	m = driveList(t, m, esc())
	if m.quit || cards(m) != "1 1a 1a1 1a2 1b 2" || cursorLine(t, m) != e.in.Cards[3].Row {
		t.Errorf("esc clears a kept filter and stays on the card: quit %v, %q", m.quit, cards(m))
	}
	if m = driveList(t, m, esc()); !m.quit {
		t.Error("esc with no filter quits")
	}
}

func TestListTree(t *testing.T) {
	e := newListEnv()
	m := driveList(t, e.start(), size(100, 20), down(), typed("t"))
	want := []string{
		"Slip Box",
		"  ├── 1  Bread  (hub, linked)",
		"> │   ├── 1a  Why does dough need time  (question, raw)",
		"  │   │   ├── 1a1  Time develops flavour  (claim, proposed)",
		"  │   │   └── 1a2  Cold slows the yeast  (claim, raw)",
		"  │   └── 1b  Salt tightens gluten  (claim, raw)",
		"  └── 2  Ovens  (idea, raw)",
	}
	if got := screen(m)[2:9]; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("t switches to the tree, on the same card:\n%s", strings.Join(got, "\n"))
	}
	if s := screen(m); !strings.Contains(s[19], "t: rows") || !strings.Contains(s[19], "fold") {
		t.Errorf("the tree's keys: %q", s[19])
	}

	// Left folds the branch, right unfolds it.
	m = driveList(t, m, key(tea.KeyLeft))
	if cards(m) != "1 1a 1b 2" || cursorLine(t, m) != "│   ├── 1a  Why does dough need time  (question, raw) [+2]" {
		t.Errorf("left folds: %s\n%s", cards(m), cursorLine(t, m))
	}
	// Left on a folded branch goes up to its parent; folding that hides all under it.
	m = driveList(t, m, typed("h"), typed("h"))
	if cards(m) != "1 2" || cursorLine(t, m) != "├── 1  Bread  (hub, linked) [+4]" {
		t.Errorf("h goes up, then folds: %s\n%s", cards(m), cursorLine(t, m))
	}
	m = driveList(t, m, typed("l"))
	if cards(m) != "1 1a 1b 2" {
		t.Errorf("l unfolds one level, and 1a stays folded: %s", cards(m))
	}
	m = driveList(t, m, down(), key(tea.KeyRight))
	if cards(m) != "1 1a 1a1 1a2 1b 2" {
		t.Errorf("right unfolds: %s", cards(m))
	}
	// Space folds and unfolds; on a leaf it does nothing, and left goes up.
	m = driveList(t, m, typed(" "))
	if cards(m) != "1 1a 1b 2" {
		t.Errorf("space folds: %s", cards(m))
	}
	m = driveList(t, m, typed(" "), down(), typed(" "))
	if cards(m) != "1 1a 1a1 1a2 1b 2" {
		t.Errorf("space unfolds, and does nothing on a leaf: %s", cards(m))
	}
	m = driveList(t, m, key(tea.KeyLeft))
	if got := cursorLine(t, m); !strings.Contains(got, "1a  Why") {
		t.Errorf("left on a leaf goes to its parent: %q", got)
	}

	// - folds every branch: the top cards are left, and each unfold opens
	// one level. + unfolds them all.
	m = driveList(t, m, typed(" "), down(), typed("-"))
	if cards(m) != "1 2" || cursorLine(t, m) != "├── 1  Bread  (hub, linked) [+4]" {
		t.Errorf("- folds all, and the cursor goes up to a card shown: %s\n%s", cards(m), cursorLine(t, m))
	}
	if m = driveList(t, m, typed("l")); cards(m) != "1 1a 1b 2" {
		t.Errorf("after fold all, unfolding opens one level: %s", cards(m))
	}
	if m = driveList(t, m, typed("+")); cards(m) != "1 1a 1a1 1a2 1b 2" {
		t.Errorf("+ unfolds all: %s", cards(m))
	}
	if m = driveList(t, driveList(t, m, typed("-")), typed("=")); cards(m) != "1 1a 1a1 1a2 1b 2" {
		t.Errorf("= unfolds all too: %s", cards(m))
	}
	m = driveList(t, m, typed("g"), down())

	// Back in the rows a fold hides nothing, and the cursor keeps its card.
	m = driveList(t, m, typed(" "), typed("t"))
	if cards(m) != "1 1a 1a1 1a2 1b 2" || cursorLine(t, m) != e.in.Cards[1].Row {
		t.Errorf("t goes back to the rows: %s", cards(m))
	}
	// A card hidden by a fold hands the cursor to the folded card.
	m = driveList(t, m, down(), typed("t"))
	if got := cursorLine(t, m); !strings.HasSuffix(got, "[+2]") {
		t.Errorf("the cursor moves to the folded branch: %q", got)
	}
}

func TestListStartsInTheTree(t *testing.T) {
	e := newListEnv()
	e.in.Tree = true
	e.in.Glyphs = tree.ASCII
	e.in.Root = "1  Bread  (hub, linked)"
	e.in.Cards = e.in.Cards[1:5]
	s := screen(driveList(t, e.start(), size(100, 20)))
	want := []string{
		"1  Bread  (hub, linked)",
		"> |-- 1a  Why does dough need time  (question, raw)",
		"  |   |-- 1a1  Time develops flavour  (claim, proposed)",
		"  |   `-- 1a2  Cold slows the yeast  (claim, raw)",
		"  `-- 1b  Salt tightens gluten  (claim, raw)",
	}
	if got := s[2:7]; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("--tree --ascii --under:\n%s", strings.Join(got, "\n"))
	}
}

// A filter in the tree keeps the matches, each under its nearest match
// above it, as zk list --tree --kind does.
func TestListFilterInTheTree(t *testing.T) {
	m := driveList(t, newListEnv().start(), size(100, 20), typed("t"), slash(), typed("claim"), enter())
	want := []string{
		"> ├── 1a1  Time develops flavour  (claim, proposed)",
		"  ├── 1a2  Cold slows the yeast  (claim, raw)",
		"  └── 1b  Salt tightens gluten  (claim, raw)",
	}
	if got := screen(m)[3:6]; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("filtered tree:\n%s", strings.Join(got, "\n"))
	}
}

func TestListOpen(t *testing.T) {
	e := newListEnv()
	m := driveList(t, e.start(), size(100, 20), down(), enter())
	if len(e.opened) != 1 || e.opened[0] != "obsidian 1a" {
		t.Errorf("enter opens the card under the cursor: %v", e.opened)
	}
	if m.quit || !strings.Contains(stripANSI(m.View()), "Opened 1a in Obsidian.") {
		t.Errorf("the list stays, and says what it opened:\n%s", stripANSI(m.View()))
	}
	if m = driveList(t, m, down()); strings.Contains(stripANSI(m.View()), "Opened 1a") {
		t.Error("the next key clears the notice")
	}

	// e opens in the editor, which gets the terminal and gives it back.
	e.opened = nil
	m = driveList(t, m, typed("e"))
	if strings.Join(e.opened, ", ") != "terminal handed over, nvim 1a1" || m.quit || m.notice != "" {
		t.Errorf("e: %v, notice %q", e.opened, m.notice)
	}

	// A failure says why and offers the editor; nothing falls back on its own.
	e.opened = nil
	e.openErr = errors.New("could not open it in Obsidian (is Obsidian installed?): open: exit status 1")
	m = driveList(t, m, enter())
	v := stripANSI(m.View())
	if len(e.opened) != 1 || !strings.Contains(v, "Could not open it in Obsidian") || !strings.Contains(v, "Press e to open it in nvim instead.") {
		t.Errorf("a failed open: %v\n%s", e.opened, v)
	}
	for _, l := range screen(m) {
		if len([]rune(l)) > 100 {
			t.Errorf("the notice is cut to the window: %q", l)
		}
	}
	if len(screen(m)) != 20 {
		t.Errorf("the notice takes a line from the cards, not from the window: %d lines", len(screen(m)))
	}

	// With no editor set, e says so.
	e.in.Editor = ""
	m = driveList(t, e.start(), enter())
	if strings.Contains(m.notice, "Press e") {
		t.Errorf("no editor to offer: %q", m.notice)
	}
	e.opened = nil
	if m = driveList(t, m, typed("e")); len(e.opened) != 0 || !strings.Contains(m.notice, "Neither $VISUAL nor $EDITOR is set") {
		t.Errorf("e with no editor: %v %q", e.opened, m.notice)
	}
}

// With --open editor, enter hands the terminal to the editor.
func TestListOpenInTheEditor(t *testing.T) {
	e := newListEnv()
	e.in.Attached, e.in.Where = true, "nvim"
	e.in.Open = e.in.Edit
	m := driveList(t, e.start(), enter())
	if strings.Join(e.opened, ", ") != "terminal handed over, nvim 1" || m.notice != "" || m.quit {
		t.Errorf("enter with --open editor: %v, notice %q", e.opened, m.notice)
	}
	e.in.Open = func(ListCard) error { return errors.New("could not open it in nvim: exit status 1") }
	m = driveList(t, e.start(), enter())
	if m.notice != "Could not open it in nvim: exit status 1." {
		t.Errorf("the editor failed, so it is not offered again: %q", m.notice)
	}
}

// The colours never change the text: a row is what zk list prints, and a
// line that is not in zk list's layout is drawn as given. The cursor's
// line has the same text, and its background runs to the window's edge.
func TestListStylingKeepsTheText(t *testing.T) {
	c := listCard("1a1", "Time develops flavour", "claim", "proposed")
	for _, p := range []palette{plainPalette, cursorPalette} {
		if got := stripANSI(p.row(c)); got != c.Row {
			t.Errorf("row: %q, want %q", got, c.Row)
		}
		if got := stripANSI(p.branch(c)); got != c.Branch {
			t.Errorf("branch: %q, want %q", got, c.Branch)
		}
	}
	odd := c
	odd.Row, odd.Branch = "1a1 | Time develops flavour", "1a1: Time develops flavour"
	if got := stripANSI(plainPalette.row(odd)); got != odd.Row {
		t.Errorf("another layout is left alone: %q", got)
	}
	if got := stripANSI(plainPalette.branch(odd)); got != odd.Branch {
		t.Errorf("another layout is left alone: %q", got)
	}

	m := driveList(t, newListEnv().start(), size(70, 10), down())
	for i, l := range strings.Split(stripANSI(m.View()), "\n")[2:8] {
		if n := len([]rune(l)); (i == 1) != (n == 70) {
			t.Errorf("only the cursor's line runs to the edge: line %d is %d wide: %q", i, n, l)
		}
	}
}

// Shift+left folds the whole family the cursor is in and goes to its top
// card; shift+right unfolds everything under the cursor.
func TestListFoldFamily(t *testing.T) {
	e := newListEnv()
	e.in.Tree = true
	e.in.Cards = append(e.in.Cards, listCard("2a", "Steam", "idea", "raw"), listCard("2a1", "Dutch ovens", "idea", "raw"))
	all := "1 1a 1a1 1a2 1b 2 2a 2a1"
	m := driveList(t, e.start(), size(100, 20), down(), down(), key(tea.KeyShiftLeft)) // on 1a1
	if cards(m) != "1 2 2a 2a1" || cursorLine(t, m) != "├── 1  Bread  (hub, linked) [+4]" {
		t.Errorf("shift+left folds the family and goes to its top: %s\n%s", cards(m), cursorLine(t, m))
	}
	if m = driveList(t, m, key(tea.KeyRight)); cards(m) != "1 1a 1b 2 2a 2a1" {
		t.Errorf("every branch of the family was folded, so right opens one level: %s", cards(m))
	}
	if m = driveList(t, m, key(tea.KeyShiftRight)); cards(m) != all {
		t.Errorf("shift+right unfolds everything under the cursor: %s", cards(m))
	}
	// H and L are the same keys. Unfolding under 1a leaves the rest folded.
	m = driveList(t, m, typed("-"), typed("l"), down(), typed("L"))
	if cards(m) != "1 1a 1a1 1a2 1b 2" {
		t.Errorf("L under 1a, with 2 still folded: %s", cards(m))
	}
	if m = driveList(t, m, down(), typed("H")); cards(m) != "1 2" || !strings.HasPrefix(cursorLine(t, m), "├── 1  Bread") {
		t.Errorf("H: %s", cards(m))
	}
	// On a top card with nothing under it, and in the rows, they do nothing.
	m = driveList(t, e.start(), typed("t"), typed("H"), typed("L"))
	if cards(m) != all {
		t.Errorf("the rows do not fold: %s", cards(m))
	}
}

// Keys read together are taken one at a time, and what follows / is the
// filter.
func TestListKeysThatArriveTogether(t *testing.T) {
	e := newListEnv()
	m := driveList(t, e.start(), size(100, 20), typed("tjj-"))
	if cards(m) != "1 2" || !strings.HasSuffix(cursorLine(t, m), "[+4]") {
		t.Errorf("tjj-: %s\n%s", cards(m), cursorLine(t, m))
	}
	if m = driveList(t, e.start(), typed("j/flav")); !m.filtering || cards(m) != "1a1" {
		t.Errorf("j/flav: filtering %v, %s", m.filtering, cards(m))
	}
	if m = driveList(t, e.start(), typed("jq")); !m.quit {
		t.Error("jq quits")
	}
}

// The keys line drops its least needed hints in a narrow window.
func TestListKeysFitTheWindow(t *testing.T) {
	e := newListEnv()
	e.in.Tree = true
	wide := screen(driveList(t, e.start(), size(120, 10)))[9]
	for _, want := range []string{"left/right: fold", "shift: family", "-/+: all", "e: editor", "q: quit"} {
		if !strings.Contains(wide, want) {
			t.Errorf("a wide window has %q: %q", want, wide)
		}
	}
	narrow := screen(driveList(t, e.start(), size(70, 10)))[9]
	if len(narrow) > 70 || strings.Contains(narrow, "e: editor") || !strings.HasSuffix(narrow, "q: quit") || !strings.Contains(narrow, "left/right: fold") {
		t.Errorf("a narrow window keeps the main keys: %q", narrow)
	}
}

func TestListQuit(t *testing.T) {
	for name, msg := range map[string]tea.Msg{"q": typed("q"), "esc": esc(), "ctrl+c": ctrlC()} {
		m := driveList(t, newListEnv().start(), msg)
		if !m.quit || m.View() != "" {
			t.Errorf("%s quits and leaves nothing drawn: quit %v, view %q", name, m.quit, m.View())
		}
	}
	if m := driveList(t, newListEnv().start(), slash(), typed("x"), ctrlC()); !m.quit {
		t.Error("ctrl+c quits while filtering")
	}
}

func TestListEmpty(t *testing.T) {
	e := newListEnv()
	e.in.Cards = nil
	m := driveList(t, e.start(), size(60, 10), down(), enter(), typed("e"), typed(" "), typed("t"), key(tea.KeyLeft))
	if v := stripANSI(m.View()); !strings.Contains(v, "0 cards") || !strings.Contains(v, "No cards.") || len(e.opened) != 0 {
		t.Errorf("an empty list:\n%s", v)
	}
}

// The parent search of zk new and the list's filter are one search.
func TestCardSearchIsShared(t *testing.T) {
	cards := []Card{
		{Key: "1", Title: "Bread", Kind: "hub", Status: "linked"},
		{Key: "2", Title: "Why does dough need time", Kind: "question", Status: "raw"},
	}
	if got := searchCards("question", cards); len(got) != 1 || got[0] != 1 {
		t.Errorf("search by kind: %v", got)
	}
	// zk new finds a parent by its kind too.
	in := input
	in.Cards = append([]Card(nil), input.Cards...)
	m := drive(t, start(in), enter(), typed("T"), enter(), slash(), typed("question"), enter(), enter())
	if got := result(m); got.Under != "2" {
		t.Errorf("zk new searched by kind and picked %+v", got)
	}
}
