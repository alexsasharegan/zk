package cli

import (
	"fmt"
	"strings"

	"github.com/alexsasharegan/zk/internal/box"
	"github.com/alexsasharegan/zk/internal/key"
)

// branches are the connector glyphs, as the tree command draws them.
type branches struct {
	mid, last, pipe, space string
}

var (
	unicodeBranches = branches{mid: "├── ", last: "└── ", pipe: "│   ", space: "    "}
	asciiBranches   = branches{mid: "|-- ", last: "`-- ", pipe: "|   ", space: "    "}
)

// cardLine is how one card reads in the tree.
func cardLine(r box.Row) string {
	return fmt.Sprintf("%s  %s  (%s, %s)", r.Key, r.Title, r.Kind, r.Status)
}

// drawTree draws cards (depth first, as zk list returns them) the way the
// tree command draws a folder: a root line, then every card hanging from
// it with connector lines, then a count. A card hangs under its nearest
// ancestor in rows, so a filtered list still draws true lines; the rest
// hang from the root.
func drawTree(root string, rows []box.Row, b branches) []string {
	shown := make(map[string]bool, len(rows))
	for _, r := range rows {
		shown[r.Key] = true
	}
	children := map[string][]box.Row{} // "" is the root
	for _, r := range rows {
		parent := ""
		for _, a := range key.Ancestors(r.Key) {
			if shown[a] {
				parent = a
				break
			}
		}
		children[parent] = append(children[parent], r)
	}

	lines := []string{root}
	var walk func(parent, prefix string)
	walk = func(parent, prefix string) {
		kids := children[parent]
		for i, r := range kids {
			branch, next := b.mid, b.pipe
			if i == len(kids)-1 {
				branch, next = b.last, b.space
			}
			lines = append(lines, prefix+branch+cardLine(r))
			walk(r.Key, prefix+next)
		}
	}
	walk("", "")

	noun := "cards"
	if len(rows) == 1 {
		noun = "card"
	}
	return append(lines, "", fmt.Sprintf("%d %s", len(rows), noun))
}

// treeText joins drawn lines for output.
func treeText(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}
