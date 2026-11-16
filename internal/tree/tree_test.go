package tree

import (
	"reflect"
	"testing"
)

func draw(keys []string, lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Prefix + keys[l.Index]
	}
	return out
}

func TestLayout(t *testing.T) {
	keys := []string{"1", "1a", "1a1", "1b", "2"}
	got := draw(keys, Layout(keys, ASCII, nil))
	want := []string{"|-- 1", "|   |-- 1a", "|   |   `-- 1a1", "|   `-- 1b", "`-- 2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// With 1a missing, 1a1 hangs under 1, its nearest ancestor present; 3a has
// none and hangs from the root.
func TestLayoutNearestAncestor(t *testing.T) {
	keys := []string{"1", "1a1", "3a"}
	lines := Layout(keys, ASCII, nil)
	got := draw(keys, lines)
	want := []string{"|-- 1", "|   `-- 1a1", "`-- 3a"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if lines[1].Parent != 0 || lines[2].Parent != -1 || lines[0].Children != 1 {
		t.Errorf("parents and children: %+v", lines)
	}
}

func TestLayoutFolded(t *testing.T) {
	keys := []string{"1", "1a", "1a1", "1b", "2"}
	lines := Layout(keys, ASCII, func(k string) bool { return k == "1" || k == "2" })
	got := draw(keys, lines)
	want := []string{"|-- 1", "`-- 2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if lines[0].Hidden != 3 || lines[1].Hidden != 0 {
		t.Errorf("hidden: %+v", lines)
	}
}
