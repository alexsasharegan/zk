// Package tree lays cards out as a tree, the way the tree command draws a
// folder. zk list --tree prints the layout, and the interactive list draws
// it with branches that fold.
package tree

import "github.com/alexsasharegan/zk/internal/key"

// Glyphs are the connector strings.
type Glyphs struct {
	Mid, Last, Pipe, Space string
}

// The two sets of connectors: box-drawing characters, and plain ones.
var (
	Unicode = Glyphs{Mid: "├── ", Last: "└── ", Pipe: "│   ", Space: "    "}
	ASCII   = Glyphs{Mid: "|-- ", Last: "`-- ", Pipe: "|   ", Space: "    "}
)

// Line is one card's place in the tree.
type Line struct {
	Index    int    // the card's index in the keys given
	Prefix   string // the connectors drawn before the card
	Parent   int    // the index of the card it hangs under, or -1 for the root
	Children int    // how many cards hang directly under it
	Hidden   int    // how many cards under it a fold hides; 0 when not folded
}

// Layout places the cards with these keys (depth first, as zk list returns
// them). A card hangs under its nearest ancestor among the keys, so a
// filtered list still draws true lines; the rest hang from the root. A card
// for which folded returns true keeps its line, and the cards under it get
// none. A nil folded folds nothing.
func Layout(keys []string, g Glyphs, folded func(key string) bool) []Line {
	index := make(map[string]int, len(keys))
	for i, k := range keys {
		index[k] = i
	}
	children := map[int][]int{} // -1 is the root
	for i, k := range keys {
		parent := -1
		for _, a := range key.Ancestors(k) {
			if j, ok := index[a]; ok {
				parent = j
				break
			}
		}
		children[parent] = append(children[parent], i)
	}

	var count func(i int) int
	count = func(i int) int {
		n := len(children[i])
		for _, c := range children[i] {
			n += count(c)
		}
		return n
	}

	var lines []Line
	var walk func(parent int, prefix string)
	walk = func(parent int, prefix string) {
		kids := children[parent]
		for n, i := range kids {
			branch, next := g.Mid, g.Pipe
			if n == len(kids)-1 {
				branch, next = g.Last, g.Space
			}
			line := Line{Index: i, Prefix: prefix + branch, Parent: parent, Children: len(children[i])}
			if folded != nil && len(children[i]) > 0 && folded(keys[i]) {
				line.Hidden = count(i)
				lines = append(lines, line)
				continue
			}
			lines = append(lines, line)
			walk(i, prefix+next)
		}
	}
	walk(-1, "")
	return lines
}
