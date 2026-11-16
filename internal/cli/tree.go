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

// drawTree draws cards (depth first, as zk list returns them) as a tree. A
// card hangs under its nearest ancestor in rows, so a filtered list still
// draws true lines; a card with none starts at the left margin.
func drawTree(rows []box.Row, b branches) []string {
	shown := make(map[string]bool, len(rows))
	for _, r := range rows {
		shown[r.Key] = true
	}
	children := map[string][]box.Row{} // "" holds the top-level cards
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

	var lines []string
	var walk func(parent, prefix string)
	walk = func(parent, prefix string) {
		kids := children[parent]
		for i, r := range kids {
			last := i == len(kids)-1
			branch, next := b.mid, b.pipe
			if last {
				branch, next = b.last, b.space
			}
			if parent == "" {
				branch, next = "", ""
			}
			lines = append(lines, fmt.Sprintf("%s%s  %s  (%s, %s)", prefix+branch, r.Key, r.Title, r.Kind, r.Status))
			walk(r.Key, prefix+next)
		}
	}
	walk("", "")
	return lines
}

// treeText joins drawn lines for output.
func treeText(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
