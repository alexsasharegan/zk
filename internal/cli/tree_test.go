package cli

import (
	"strings"
	"testing"

	"github.com/alexsasharegan/zk/internal/box"
	"github.com/alexsasharegan/zk/internal/key"
)

func rows(keys ...string) []box.Row {
	out := make([]box.Row, len(keys))
	for i, k := range keys {
		out[i] = box.Row{Key: k, Level: key.Level(k), Title: "T" + k, Kind: "idea", Status: "raw"}
	}
	return out
}

func TestDrawTree(t *testing.T) {
	got := treeText(drawTree(rows("1", "1a", "1a1", "1a2", "1b", "1b1", "2", "2a"), unicodeBranches))
	want := `1  T1  (idea, raw)
├── 1a  T1a  (idea, raw)
│   ├── 1a1  T1a1  (idea, raw)
│   └── 1a2  T1a2  (idea, raw)
└── 1b  T1b  (idea, raw)
    └── 1b1  T1b1  (idea, raw)
2  T2  (idea, raw)
└── 2a  T2a  (idea, raw)
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDrawTreeASCII(t *testing.T) {
	got := treeText(drawTree(rows("1", "1a", "1a1", "1b"), asciiBranches))
	want := "1  T1  (idea, raw)\n|-- 1a  T1a  (idea, raw)\n|   `-- 1a1  T1a1  (idea, raw)\n`-- 1b  T1b  (idea, raw)\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// With 1a filtered out, its children hang under 1, the nearest shown
// ancestor; a card with no shown ancestor starts at the margin.
func TestDrawTreeFiltered(t *testing.T) {
	got := treeText(drawTree(rows("1", "1a1", "1a2", "2a", "2a1"), unicodeBranches))
	want := `1  T1  (idea, raw)
├── 1a1  T1a1  (idea, raw)
└── 1a2  T1a2  (idea, raw)
2a  T2a  (idea, raw)
└── 2a1  T2a1  (idea, raw)
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if treeText(drawTree(nil, unicodeBranches)) != "" {
		t.Error("an empty list draws nothing")
	}
}

func TestListTreeCommand(t *testing.T) {
	b := newBox(t, defaultSpec)
	b.ok("new", "idea", "One")
	b.ok("new", "claim", "A", "--under", "1")
	b.ok("new", "idea", "B", "--under", "1")
	out := b.ok("ls", "--tree")
	want := "1  One  (idea, raw)\n├── 1a  A  (claim, raw)\n└── 1b  B  (idea, raw)\n"
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
	if out := b.ok("ls", "--tree", "--ascii", "--under", "1"); out != "1  One  (idea, raw)\n|-- 1a  A  (claim, raw)\n`-- 1b  B  (idea, raw)\n" {
		t.Errorf("--tree --under 1 is rooted at 1:\n%s", out)
	}
	if out := b.ok("ls", "--under", "1"); strings.Count(out, "\n") != 2 {
		t.Errorf("plain --under lists only the descendants:\n%s", out)
	}
	if out := b.ok("ls", "--tree", "--json"); strings.Count(out, "\n") != 3 || !strings.Contains(out, `"key":"1a"`) {
		t.Errorf("--json wins over --tree:\n%s", out)
	}
}
