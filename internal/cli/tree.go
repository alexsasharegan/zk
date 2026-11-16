package cli

import (
	"fmt"
	"strings"

	"github.com/alexsasharegan/zk/internal/box"
	"github.com/alexsasharegan/zk/internal/tree"
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
func drawTree(root string, rows []box.Row, g tree.Glyphs) []string {
	keys := make([]string, len(rows))
	for i, r := range rows {
		keys[i] = r.Key
	}
	lines := []string{root}
	for _, l := range tree.Layout(keys, g, nil) {
		lines = append(lines, l.Prefix+cardLine(rows[l.Index]))
	}

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
