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

// Like the tree command: one root line, every card hanging from it, roots
// included, then a count.
func TestDrawTree(t *testing.T) {
	got := treeText(drawTree("Slip Box", rows("1", "1a", "1a1", "1a2", "1b", "1b1", "2", "2a"), unicodeBranches))
	want := `Slip Box
├── 1  T1  (idea, raw)
│   ├── 1a  T1a  (idea, raw)
│   │   ├── 1a1  T1a1  (idea, raw)
│   │   └── 1a2  T1a2  (idea, raw)
│   └── 1b  T1b  (idea, raw)
│       └── 1b1  T1b1  (idea, raw)
└── 2  T2  (idea, raw)
    └── 2a  T2a  (idea, raw)

8 cards
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDrawTreeASCII(t *testing.T) {
	got := treeText(drawTree("Box", rows("1", "1a", "2"), asciiBranches))
	want := "Box\n|-- 1  T1  (idea, raw)\n|   `-- 1a  T1a  (idea, raw)\n`-- 2  T2  (idea, raw)\n\n3 cards\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// With 1a filtered out, its children hang under 1, the nearest shown
// ancestor; a card with none shown hangs from the root.
func TestDrawTreeFiltered(t *testing.T) {
	got := treeText(drawTree("Box", rows("1", "1a1", "1a2", "2a", "2a1"), unicodeBranches))
	want := `Box
├── 1  T1  (idea, raw)
│   ├── 1a1  T1a1  (idea, raw)
│   └── 1a2  T1a2  (idea, raw)
└── 2a  T2a  (idea, raw)
    └── 2a1  T2a1  (idea, raw)

5 cards
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := treeText(drawTree("Box", nil, unicodeBranches)); got != "Box\n\n0 cards\n" {
		t.Errorf("empty: %q", got)
	}
	if got := treeText(drawTree("Box", rows("1"), unicodeBranches)); !strings.HasSuffix(got, "\n1 card\n") {
		t.Errorf("one card: %q", got)
	}
}

func TestListTreeCommand(t *testing.T) {
	b := newBox(t, defaultSpec)
	b.ok("new", "idea", "One")
	b.ok("new", "claim", "A", "--under", "1")
	b.ok("new", "idea", "B", "--under", "1")
	b.ok("new", "idea", "Two")
	want := "Slip Box\n├── 1  One  (idea, raw)\n│   ├── 1a  A  (claim, raw)\n│   └── 1b  B  (idea, raw)\n└── 2  Two  (idea, raw)\n\n4 cards\n"
	if out := b.ok("ls", "--tree"); out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
	want = "1  One  (idea, raw)\n|-- 1a  A  (claim, raw)\n`-- 1b  B  (idea, raw)\n\n2 cards\n"
	if out := b.ok("ls", "--tree", "--ascii", "--under", "1"); out != want {
		t.Errorf("--tree --under 1 is rooted at 1:\n%s\nwant:\n%s", out, want)
	}
	if out := b.ok("ls", "--under", "1"); strings.Count(out, "\n") != 2 {
		t.Errorf("plain --under lists only the descendants:\n%s", out)
	}
	if out := b.ok("ls", "--tree", "--json"); strings.Count(out, "\n") != 4 || !strings.Contains(out, `"key":"1a"`) {
		t.Errorf("--json wins over --tree:\n%s", out)
	}
}
