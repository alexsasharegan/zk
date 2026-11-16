// Package tui holds zk's terminal UIs. The first is zk new: pick a kind,
// write a title, choose a parent (ADR 261115-czb2).
package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Kind is one kind of card, as the spec describes it.
type Kind struct {
	Name, Description string
}

// Card is an existing card, offered as a parent.
type Card struct {
	Key, Title, Kind string
	Level            int
}

// Input is what the UI needs: the box, its kinds and cards, and anything
// already given on the command line.
type Input struct {
	Box    string // the box's name
	Reason string // why this box was chosen
	Kinds  []Kind
	Cards  []Card // depth first
	Kind   string // given on the command line, or ""
	Title  string // given on the command line, or ""
	Under  string // given with --under, or ""
	// AskParent: offer the parent step (false when --under was given).
	AskParent bool
}

// Result is what the user chose.
type Result struct {
	Kind, Title, Under string
	Cancelled          bool
}

type step int

const (
	stepKind step = iota
	stepTitle
	stepParent
	stepDone
)

// Model is the zk new flow. It is a plain Bubble Tea model, so tests can
// drive it with key messages and no terminal.
type Model struct {
	in      Input
	step    step
	kinds   list.Model
	title   textinput.Model
	parents list.Model
	result  Result
	width   int
	height  int
	notice  string
}

type kindItem struct{ k Kind }

func (i kindItem) Title() string       { return i.k.Name }
func (i kindItem) Description() string { return i.k.Description }
func (i kindItem) FilterValue() string { return i.k.Name }

type parentItem struct {
	card Card
	root bool
}

func (p parentItem) Title() string {
	if p.root {
		return "No parent: a new root"
	}
	return strings.Repeat("  ", p.card.Level-1) + p.card.Key + "  " + p.card.Title
}

func (p parentItem) Description() string {
	if p.root {
		return "Start a new thread"
	}
	return strings.Repeat("  ", p.card.Level-1) + p.card.Kind
}

func (p parentItem) FilterValue() string {
	if p.root {
		return "new root"
	}
	return p.card.Key + " " + p.card.Title
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	promptStyle = lipgloss.NewStyle().Bold(true)
)

// NewModel builds the flow, starting at the first step the command line
// left open.
func NewModel(in Input) Model {
	m := Model{in: in, width: 80, height: 24}
	m.result = Result{Kind: in.Kind, Title: in.Title, Under: in.Under}

	items := make([]list.Item, len(in.Kinds))
	selected := 0
	for i, k := range in.Kinds {
		items[i] = kindItem{k}
		if k.Name == in.Kind {
			selected = i
		}
	}
	m.kinds = newList(items, "What kind of card?")
	m.kinds.Select(selected)

	m.title = textinput.New()
	m.title.Placeholder = "a short label, a noun phrase"
	m.title.SetValue(in.Title)
	m.title.CharLimit = 200

	parents := []list.Item{parentItem{root: true}}
	for _, c := range in.Cards {
		parents = append(parents, parentItem{card: c})
	}
	m.parents = newList(parents, "Under which card? (/ to search)")

	switch {
	case in.Kind == "":
		m.step = stepKind
	case in.Title == "":
		m.step = stepTitle
		m.title.Focus()
	case in.AskParent:
		m.step = stepParent
	default:
		m.step = stepDone
	}
	m.resize()
	return m
}

func newList(items []list.Item, title string) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.DisableQuitKeybindings()
	return l
}

func (m *Model) resize() {
	h := m.height - 3 // header and notice lines
	if h < 6 {
		h = 6
	}
	m.kinds.SetSize(m.width, h)
	m.parents.SetSize(m.width, h)
	m.title.Width = m.width - 4
}

// Init starts the cursor blinking when the flow opens on the title.
func (m Model) Init() tea.Cmd {
	if m.step == stepTitle {
		return textinput.Blink
	}
	return nil
}

// Result is the user's choice; valid once the program has quit.
func (m Model) Result() Result { return m.result }

// Done reports whether the flow has finished, by choice or by cancelling.
func (m Model) Done() bool { return m.step == stepDone || m.result.Cancelled }

// Update handles a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.result.Cancelled = true
			return m, tea.Quit
		}
	}
	switch m.step {
	case stepKind:
		return m.updateKind(msg)
	case stepTitle:
		return m.updateTitle(msg)
	case stepParent:
		return m.updateParent(msg)
	}
	return m, nil
}

func (m Model) updateKind(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.kinds.FilterState() == list.Unfiltered {
		switch key.Type {
		case tea.KeyEnter:
			if it, ok := m.kinds.SelectedItem().(kindItem); ok {
				m.result.Kind = it.k.Name
				m.step = stepTitle
				m.notice = ""
				m.title.Focus()
				return m, textinput.Blink
			}
		case tea.KeyEsc:
			m.result.Cancelled = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.kinds, cmd = m.kinds.Update(msg)
	return m, cmd
}

func (m Model) updateTitle(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			t := strings.Join(strings.Fields(m.title.Value()), " ")
			if t == "" {
				m.notice = "A card needs a title."
				return m, nil
			}
			m.result.Title = t
			m.title.Blur()
			m.notice = ""
			if !m.in.AskParent {
				m.step = stepDone
				return m, tea.Quit
			}
			m.step = stepParent
			return m, nil
		case tea.KeyEsc:
			m.title.Blur()
			m.step = stepKind
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.title, cmd = m.title.Update(msg)
	return m, cmd
}

func (m Model) updateParent(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.parents.FilterState() != list.Filtering {
		switch key.Type {
		case tea.KeyEnter:
			if it, ok := m.parents.SelectedItem().(parentItem); ok {
				if it.root {
					m.result.Under = ""
				} else {
					m.result.Under = it.card.Key
				}
				m.step = stepDone
				return m, tea.Quit
			}
		case tea.KeyEsc:
			if m.parents.FilterState() == list.FilterApplied {
				break // let the list clear its filter
			}
			m.step = stepTitle
			m.title.Focus()
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	m.parents, cmd = m.parents.Update(msg)
	return m, cmd
}

// View draws the flow.
func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.header() + "\n")
	switch m.step {
	case stepKind:
		b.WriteString(m.kinds.View())
	case stepTitle:
		fmt.Fprintf(&b, "\n%s\n\n", promptStyle.Render("Title for a new "+m.result.Kind+":"))
		b.WriteString("  " + m.title.View() + "\n\n")
		if k, ok := m.kind(m.result.Kind); ok {
			b.WriteString(dimStyle.Render("  "+k.Description) + "\n")
		}
		b.WriteString(dimStyle.Render("  enter: next   esc: back   ctrl+c: cancel") + "\n")
	case stepParent:
		b.WriteString(dimStyle.Render("  "+m.result.Kind+": "+m.result.Title) + "\n")
		b.WriteString(m.parents.View())
	}
	if m.notice != "" {
		b.WriteString("\n" + m.notice + "\n")
	}
	return b.String()
}

// header names the box and why it was chosen, cut to the window's width.
func (m Model) header() string {
	reason := "  (" + m.in.Reason + ")"
	room := m.width - len("zk new  ") - len([]rune(m.in.Box))
	if r := []rune(reason); room < len(r) {
		if room > 4 {
			reason = string(r[:room-4]) + "...)"
		} else {
			reason = ""
		}
	}
	return headerStyle.Render("zk new") + "  " + m.in.Box + dimStyle.Render(reason)
}

func (m Model) kind(name string) (Kind, bool) {
	for _, k := range m.in.Kinds {
		if k.Name == name {
			return k, true
		}
	}
	return Kind{}, false
}

// Run runs the flow on a terminal and returns the user's choice.
func Run(in Input, input io.Reader, output io.Writer) (Result, error) {
	m := NewModel(in)
	if m.Done() {
		return m.Result(), nil
	}
	final, err := tea.NewProgram(m, tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen()).Run()
	if err != nil {
		return Result{}, err
	}
	return final.(Model).Result(), nil
}
