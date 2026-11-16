package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

// The card search is one piece, shared by the parent step of zk new and the
// filter of the interactive list: a fuzzy match of what was typed against a
// card's key, title, kind and status.

// searchText is the text of a card the search matches.
func searchText(c Card) string {
	return strings.Join(strings.Fields(c.Key+" "+c.Title+" "+c.Kind+" "+c.Status), " ")
}

// cardFilter ranks the search texts that match a term. It is the filter of
// a bubbles list, so a list of cards searches the same way.
var cardFilter list.FilterFunc = list.DefaultFilter

// searchCards returns the indexes of the cards that match term, in the
// order the cards were given. An empty term matches every card.
func searchCards(term string, cards []Card) []int {
	out := make([]int, 0, len(cards))
	if strings.TrimSpace(term) == "" {
		for i := range cards {
			out = append(out, i)
		}
		return out
	}
	texts := make([]string, len(cards))
	for i, c := range cards {
		texts[i] = searchText(c)
	}
	for _, r := range cardFilter(term, texts) {
		out = append(out, r.Index)
	}
	sort.Ints(out)
	return out
}
