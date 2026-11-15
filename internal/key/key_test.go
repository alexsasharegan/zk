package key

import (
	"reflect"
	"sort"
	"testing"
)

func TestValid(t *testing.T) {
	for _, k := range []string{"1", "7", "10", "7a", "7a1", "7aa", "7a1a", "123zz45"} {
		if !Valid(k) {
			t.Errorf("%s should be valid", k)
		}
	}
	for _, k := range []string{"", "0", "07", "0x", "a", "7A", "7a0", "7a01", "7-a", "zk-7"} {
		if Valid(k) {
			t.Errorf("%s should be invalid", k)
		}
	}
}

// The cases from the slip tool's tests.
func TestSortAndLevel(t *testing.T) {
	cases := map[string]struct {
		sort  string
		level int
	}{
		"7a1": {"00000007/00000001/00000001", 3},
		"7aa": {"00000007/00000027", 2},
		"10":  {"00000010", 1},
		"2a":  {"00000002/00000001", 2},
	}
	for k, want := range cases {
		if got := Sort(k); got != want.sort {
			t.Errorf("Sort(%s) = %s, want %s", k, got, want.sort)
		}
		if got := Level(k); got != want.level {
			t.Errorf("Level(%s) = %d, want %d", k, got, want.level)
		}
	}
}

func TestLetters(t *testing.T) {
	for n, s := range map[int]string{1: "a", 26: "z", 27: "aa", 52: "az", 53: "ba", 702: "zz", 703: "aaa"} {
		if got := Letters(n); got != s {
			t.Errorf("Letters(%d) = %s, want %s", n, got, s)
		}
		if got := LettersValue(s); got != n {
			t.Errorf("LettersValue(%s) = %d, want %d", s, got, n)
		}
	}
}

func TestDepthFirstOrder(t *testing.T) {
	keys := []string{"3", "2b2", "1a", "2", "10", "2aa", "2b", "1", "2a", "1a1", "2b1", "2z"}
	sort.Slice(keys, func(i, j int) bool { return Less(keys[i], keys[j]) })
	want := []string{"1", "1a", "1a1", "2", "2a", "2b", "2b1", "2b2", "2z", "2aa", "3", "10"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("got %v\nwant %v", keys, want)
	}
}

func TestFamily(t *testing.T) {
	if Parent("7a1") != "7a" || Parent("7a") != "7" || Parent("7") != "" {
		t.Error("Parent")
	}
	if got := Ancestors("7a1a"); !reflect.DeepEqual(got, []string{"7a1", "7a", "7"}) {
		t.Errorf("Ancestors: %v", got)
	}
	if !IsAncestor("7", "7a1") || IsAncestor("7a1", "7a") || IsAncestor("70", "7a") {
		t.Error("IsAncestor")
	}
	if Tag("zk", "7a1") != "zk/7/a/1" || Stem("zk", "7a1") != "zk-7a1" {
		t.Error("Tag or Stem")
	}
	if FromStem("zk", "zk-7a1") != "7a1" || FromStem("zk", "zkx-1") != "" || FromStem("zk", "notes") != "" {
		t.Error("FromStem")
	}
	for k, want := range map[string]string{"5": "4", "1": "", "2b": "2a", "2a": "", "7aa": "7z", "7a10": "7a9"} {
		if got := Previous(k); got != want {
			t.Errorf("Previous(%s) = %q, want %q", k, got, want)
		}
	}
}

// The order the slip tool's tests make cards in.
func TestNext(t *testing.T) {
	var keys []string
	add := func(k string) string { keys = append(keys, k); return k }
	steps := []struct{ got, want string }{
		{add(NextRoot(keys)), "1"},
		{add(NextRoot(keys)), "2"},
		{add(NextChild("2", keys)), "2a"},
		{add(NextChild("2", keys)), "2b"},
		{add(NextChild("2a", keys)), "2a1"},
		{add(NextChild("2a1", keys)), "2a1a"},
		{add(NextRoot(keys)), "3"},
		{NextChild("2a", keys), "2a2"},
	}
	for i, s := range steps {
		if s.got != s.want {
			t.Errorf("step %d: got %s, want %s", i, s.got, s.want)
		}
	}
	// A deleted key is not reused: the next child follows the highest.
	if got := NextChild("2", []string{"2", "2a", "2c"}); got != "2d" {
		t.Errorf("after a gap: %s", got)
	}
	if got := NextRoot([]string{"1", "9", "10", "10a"}); got != "11" {
		t.Errorf("NextRoot: %s", got)
	}
}
