package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	cardkey "github.com/alexsasharegan/zk/internal/key"
	"github.com/alexsasharegan/zk/internal/tree"
)

// ListCard is one card of the interactive list, with the text zk list
// prints for it.
type ListCard struct {
	Card
	Row    string // its line in the rows, as zk list prints it
	Branch string // its text in the tree, as zk list --tree prints it
}

// ListInput is what the interactive list needs (ADR 261115-9pqw).
type ListInput struct {
	Box    string // the box's name
	Reason string // why this box was chosen
	Root   string // the tree's first line: the box, or the --under card
	Cards  []ListCard
	Tree   bool        // start in the tree
	Glyphs tree.Glyphs // the tree's connectors

	// Open opens a card where --open says; Where names the place for the
	// notice ("Obsidian"). Attached means Open needs the terminal, as an
	// editor does.
	Open     func(ListCard) error
	Where    string
	Attached bool
	// Edit opens a card in the editor, named Editor; Editor is "" when
	// none is set.
	Edit   func(ListCard) error
	Editor string
}

// ListModel is the interactive list. Like Model, it is a plain Bubble Tea
// model that tests drive with key messages.
type ListModel struct {
	in        ListInput
	tree      bool
	folded    map[string]bool
	filter    textinput.Model
	filtering bool
	matches   []int      // the cards the filter keeps, in the box's order
	lines     []listLine // what is drawn, top to bottom
	cursor    int        // index into lines
	offset    int        // the first line drawn
	width     int
	height    int
	notice    string
	quit      bool

	// exec runs fn with the terminal handed back, then reports. Tests
	// replace it.
	exec func(fn func() error, done func(error) tea.Msg) tea.Cmd
}

// listLine is one drawn line: a card, and in the tree its place.
type listLine struct {
	card     int    // index into in.Cards
	parent   int    // the card it hangs under, or -1
	prefix   string // the tree's connectors
	children int
	hidden   int // cards a fold hides under it
}

// openedMsg reports that a card was opened, or why it was not.
type openedMsg struct {
	card   ListCard
	where  string
	editor bool
	err    error
}

var selectedStyle = lipgloss.NewStyle().Reverse(true)

// NewListModel builds the interactive list.
func NewListModel(in ListInput) ListModel {
	m := ListModel{in: in, tree: in.Tree, folded: map[string]bool{}, width: 80, height: 24}
	m.filter = textinput.New()
	m.filter.Prompt = "/"
	m.filter.CharLimit = 200
	m.filter.Width = m.width - 24 // room for the count
	m.exec = func(fn func() error, done func(error) tea.Msg) tea.Cmd {
		return tea.Exec(&execFunc{fn: fn}, done)
	}
	m.layout()
	return m
}

// execFunc lets Bubble Tea run a function with the terminal released. The
// function uses the process's own stdin and stdout, so the streams Bubble
// Tea offers are not needed.
type execFunc struct{ fn func() error }

func (e *execFunc) Run() error          { return e.fn() }
func (e *execFunc) SetStdin(io.Reader)  {}
func (e *execFunc) SetStdout(io.Writer) {}
func (e *execFunc) SetStderr(io.Writer) {}

// Init does nothing; the list is drawn as it stands.
func (m ListModel) Init() tea.Cmd { return nil }

// layout works out the lines to draw from the filter, the view and the
// folds, and keeps the cursor on its card, or on the nearest card above it
// in the tree when its own line is gone.
func (m *ListModel) layout() {
	at := ""
	if m.cursor < len(m.lines) {
		at = m.in.Cards[m.lines[m.cursor].card].Key
	}

	cards := make([]Card, len(m.in.Cards))
	for i, c := range m.in.Cards {
		cards[i] = c.Card
	}
	m.matches = searchCards(m.filter.Value(), cards)

	m.lines = m.lines[:0]
	if m.tree {
		keys := make([]string, len(m.matches))
		for i, c := range m.matches {
			keys[i] = m.in.Cards[c].Key
		}
		for _, l := range tree.Layout(keys, m.in.Glyphs, func(k string) bool { return m.folded[k] }) {
			parent := -1
			if l.Parent >= 0 {
				parent = m.matches[l.Parent]
			}
			m.lines = append(m.lines, listLine{card: m.matches[l.Index], parent: parent, prefix: l.Prefix, children: l.Children, hidden: l.Hidden})
		}
	} else {
		for _, c := range m.matches {
			m.lines = append(m.lines, listLine{card: c, parent: -1})
		}
	}

	m.cursor = 0
	if at != "" {
		where := map[string]int{}
		for i, l := range m.lines {
			where[m.in.Cards[l.card].Key] = i
		}
		for _, k := range append([]string{at}, cardkey.Ancestors(at)...) {
			if i, ok := where[k]; ok {
				m.cursor = i
				break
			}
		}
	}
	m.scroll()
}

// body is how many lines of cards fit in the window.
func (m ListModel) body() int {
	h := m.height - 3 // the header, the count line and the keys
	if m.tree {
		h-- // the root line
	}
	if m.notice != "" {
		h -= strings.Count(m.notice, "\n") + 1
	}
	if h < 1 {
		h = 1
	}
	return h
}

// scroll keeps the cursor in range and in the window.
func (m *ListModel) scroll() {
	if m.cursor >= len(m.lines) {
		m.cursor = len(m.lines) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	h := m.body()
	if m.offset > len(m.lines)-h {
		m.offset = len(m.lines) - h
	}
	if m.offset > m.cursor {
		m.offset = m.cursor
	}
	if m.offset < m.cursor-h+1 {
		m.offset = m.cursor - h + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// selected is the card under the cursor.
func (m ListModel) selected() (listLine, bool) {
	if m.cursor < 0 || m.cursor >= len(m.lines) {
		return listLine{}, false
	}
	return m.lines[m.cursor], true
}

// Update handles a message.
func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.filter.Width = m.width - 24 // room for the count
		m.scroll()
		return m, nil
	case openedMsg:
		m.notice = m.openedNotice(msg)
		m.scroll()
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quit = true
			return m, tea.Quit
		}
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateKey(msg)
	}
	return m, nil
}

// updateFilter handles a key while the filter is being typed.
func (m ListModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.filtering = false
		m.filter.Blur()
		return m, nil
	case tea.KeyEsc:
		m.filtering = false
		m.filter.Blur()
		m.filter.SetValue("")
		m.layout()
		return m, nil
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
		m.move(msg)
		return m, nil
	}
	before := m.filter.Value()
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != before {
		m.layout()
	}
	return m, cmd
}

// move handles the keys that move the cursor, and reports whether msg was
// one of them.
func (m *ListModel) move(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "pgup":
		m.cursor -= m.body()
	case "pgdown":
		m.cursor += m.body()
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(m.lines) - 1
	default:
		return false
	}
	m.scroll()
	return true
}

// updateKey handles a key while the list has the keyboard.
func (m ListModel) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notice = ""
	if m.move(msg) {
		return m, nil
	}
	line, ok := m.selected()
	switch msg.String() {
	case "q":
		m.quit = true
		return m, tea.Quit
	case "esc":
		if m.filter.Value() != "" {
			m.filter.SetValue("")
			m.layout()
			return m, nil
		}
		m.quit = true
		return m, tea.Quit
	case "/":
		m.filtering = true
		m.filter.CursorEnd()
		return m, m.filter.Focus()
	case "t":
		m.tree = !m.tree
		m.layout()
		return m, nil
	case "left", "h":
		if !m.tree || !ok {
			break
		}
		if line.children > 0 && line.hidden == 0 {
			m.folded[m.in.Cards[line.card].Key] = true
			m.layout()
			break
		}
		for i, l := range m.lines { // already folded, or a leaf: go up
			if l.card == line.parent {
				m.cursor = i
			}
		}
	case "right", "l":
		if m.tree && ok && line.hidden > 0 {
			delete(m.folded, m.in.Cards[line.card].Key)
			m.layout()
		}
	case " ":
		if m.tree && ok && line.children > 0 {
			k := m.in.Cards[line.card].Key
			if m.folded[k] {
				delete(m.folded, k)
			} else {
				m.folded[k] = true
			}
			m.layout()
		}
	case "enter":
		if ok && m.in.Open != nil {
			return m, m.open(m.in.Cards[line.card], false)
		}
	case "e":
		if !ok {
			break
		}
		if m.in.Editor == "" || m.in.Edit == nil {
			m.notice = "Neither $VISUAL nor $EDITOR is set, so there is no editor to open it in."
			break
		}
		return m, m.open(m.in.Cards[line.card], true)
	}
	m.scroll()
	return m, nil
}

// open opens a card: where --open says, or in the editor. An editor gets
// the terminal and gives it back; Obsidian is opened beside the list.
func (m ListModel) open(c ListCard, editor bool) tea.Cmd {
	fn, where, attached := m.in.Open, m.in.Where, m.in.Attached
	if editor {
		fn, where, attached = m.in.Edit, m.in.Editor, true
	}
	editor = editor || attached
	done := func(err error) tea.Msg { return openedMsg{card: c, where: where, editor: editor, err: err} }
	if attached {
		return m.exec(func() error { return fn(c) }, done)
	}
	return func() tea.Msg { return done(fn(c)) }
}

// openedNotice says what opening a card did. A failure says why, and
// offers the editor when the card was not opened there (ADR 261115-mjrm:
// nothing falls back on its own).
func (m ListModel) openedNotice(msg openedMsg) string {
	if msg.err == nil {
		if msg.editor {
			return "" // the editor was on the screen; nothing to add
		}
		return fmt.Sprintf("Opened %s in %s.", msg.card.Key, msg.where)
	}
	notice := upperFirst(msg.err.Error()) + "."
	if !msg.editor && m.in.Editor != "" {
		notice += "\nPress e to open it in " + m.in.Editor + " instead."
	}
	return notice
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// View draws the list.
func (m ListModel) View() string {
	if m.quit {
		return ""
	}
	var b strings.Builder
	b.WriteString(header("zk list", m.in.Box, m.in.Reason, m.width) + "\n")
	b.WriteString(m.countLine() + "\n")
	if m.tree {
		b.WriteString(m.fit(m.in.Root) + "\n")
	}
	h := m.body()
	for i := m.offset; i < m.offset+h; i++ {
		if i < len(m.lines) {
			b.WriteString(m.line(i))
		} else if i == 0 {
			b.WriteString(dimStyle.Render(m.empty()))
		}
		b.WriteString("\n")
	}
	if m.notice != "" {
		for _, l := range strings.Split(m.notice, "\n") {
			b.WriteString(m.fit(l) + "\n")
		}
	}
	b.WriteString(dimStyle.Render(m.fit(m.keys())))
	return b.String()
}

// fit cuts a line to the window's width.
func (m ListModel) fit(s string) string {
	return lipgloss.NewStyle().Inline(true).MaxWidth(m.width).Render(s)
}

// line draws one card, marked when the cursor is on it.
func (m ListModel) line(i int) string {
	l := m.lines[i]
	c := m.in.Cards[l.card]
	text := c.Row
	if m.tree {
		text = l.prefix + c.Branch
		if l.hidden > 0 {
			text += fmt.Sprintf(" [+%d]", l.hidden)
		}
	}
	if i == m.cursor {
		return selectedStyle.Render(m.fit("> " + text))
	}
	return m.fit("  " + text)
}

// empty is what the list says in place of cards.
func (m ListModel) empty() string {
	if m.filter.Value() != "" {
		return "  No card matches the filter."
	}
	return "  No cards."
}

// countLine is the filter, while it is typed or set, and how many cards
// are listed.
func (m ListModel) countLine() string {
	count := fmt.Sprintf("%d cards", len(m.in.Cards))
	if len(m.in.Cards) == 1 {
		count = "1 card"
	}
	if m.filter.Value() != "" {
		count = fmt.Sprintf("%d of %s", len(m.matches), count)
	}
	switch {
	case m.filtering:
		return m.filter.View() + "  " + dimStyle.Render(count)
	case m.filter.Value() != "":
		return m.fit("/" + m.filter.Value() + "  " + dimStyle.Render(count))
	}
	return dimStyle.Render(count)
}

// keys is the line of keys at the bottom.
func (m ListModel) keys() string {
	if m.filtering {
		return "enter: keep the filter   esc: clear it"
	}
	keys := "/: filter   t: tree   "
	if m.tree {
		keys = "/: filter   t: rows   left/right/space: fold   "
	}
	keys += "enter: open   e: editor   "
	if m.filter.Value() != "" {
		return keys + "esc: clear the filter   q: quit"
	}
	return keys + "q: quit"
}

// RunList runs the interactive list on a terminal until the user quits.
func RunList(in ListInput, input io.Reader, output io.Writer) error {
	_, err := tea.NewProgram(NewListModel(in), tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen()).Run()
	return err
}
