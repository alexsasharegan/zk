// Package key handles card keys: permanent addresses such as 7a1, made of
// segments that alternate between numbers and letters.
//
// Level 1 is a number, level 2 a letter, and so on. Numbers are positive
// with no leading zero; letters count like spreadsheet columns (a to z, then
// aa). A key is never changed or reused.
package key

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	valid   = regexp.MustCompile(`^[1-9][0-9]*(?:[a-z]+[1-9][0-9]*)*(?:[a-z]+)?$`)
	segment = regexp.MustCompile(`[0-9]+|[a-z]+`)
)

// Valid reports whether k is a well-formed key.
func Valid(k string) bool { return valid.MatchString(k) }

// Segments splits a key where digits meet letters: 7a1 is 7, a, 1.
func Segments(k string) []string { return segment.FindAllString(k, -1) }

// Level is the number of segments.
func Level(k string) int { return len(Segments(k)) }

// Parent is the key without its last segment, or "" for a root.
func Parent(k string) string {
	s := Segments(k)
	if len(s) <= 1 {
		return ""
	}
	return strings.Join(s[:len(s)-1], "")
}

// Ancestors are the parent, its parent, and so on up to the root.
func Ancestors(k string) []string {
	var out []string
	for p := Parent(k); p != ""; p = Parent(p) {
		out = append(out, p)
	}
	return out
}

// IsAncestor reports whether a is an ancestor of k.
func IsAncestor(a, k string) bool {
	for _, p := range Ancestors(k) {
		if p == a {
			return true
		}
	}
	return false
}

// Letters turns 1 into a, 26 into z, 27 into aa.
func Letters(n int) string {
	var b []byte
	for n > 0 {
		n--
		b = append([]byte{byte('a' + n%26)}, b...)
		n /= 26
	}
	return string(b)
}

// LettersValue turns a into 1, z into 26, aa into 27.
func LettersValue(s string) int {
	n := 0
	for _, c := range s {
		n = n*26 + int(c-'a') + 1
	}
	return n
}

// value is a segment's number: its value, or its spreadsheet index.
func value(seg string) int {
	if n, err := strconv.Atoi(seg); err == nil {
		return n
	}
	return LettersValue(seg)
}

// Sort is the sort key: each segment's number padded to 8 digits, joined
// with "/", so depth-first order is plain text order.
func Sort(k string) string {
	s := Segments(k)
	out := make([]string, len(s))
	for i, seg := range s {
		out[i] = fmt.Sprintf("%08d", value(seg))
	}
	return strings.Join(out, "/")
}

// Tag is the key's Obsidian tag: the prefix and every segment, zk/7/a/1.
func Tag(prefix, k string) string {
	return prefix + "/" + strings.Join(Segments(k), "/")
}

// Stem is the card's file name without .md, and its wikilink target.
func Stem(prefix, k string) string { return prefix + "-" + k }

// FromStem returns the key in a file stem or link target, or "" when the
// name is not in the prefix's namespace.
func FromStem(prefix, stem string) string {
	if !strings.HasPrefix(stem, prefix+"-") {
		return ""
	}
	return stem[len(prefix)+1:]
}

// Less orders keys depth first.
func Less(a, b string) bool { return Sort(a) < Sort(b) }

// Previous returns the sibling before k ("" for a first child or root 1),
// used to spot a skipped key.
func Previous(k string) string {
	s := Segments(k)
	last := s[len(s)-1]
	n := value(last)
	if n <= 1 {
		return ""
	}
	var prev string
	if last[0] >= '0' && last[0] <= '9' {
		prev = strconv.Itoa(n - 1)
	} else {
		prev = Letters(n - 1)
	}
	return Parent(k) + prev
}

// NextRoot is the next unused root: one more than the highest.
func NextRoot(keys []string) string {
	top := 0
	for _, k := range keys {
		if Level(k) == 1 {
			if n := value(k); n > top {
				top = n
			}
		}
	}
	return strconv.Itoa(top + 1)
}

// NextChild is the first free child of parent: one past its highest child,
// a letter under a number and a number under a letter.
func NextChild(parent string, keys []string) string {
	wantNumber := (Level(parent)+1)%2 == 1
	top := 0
	for _, k := range keys {
		if Parent(k) == parent && Level(k) == Level(parent)+1 {
			s := Segments(k)
			if n := value(s[len(s)-1]); n > top {
				top = n
			}
		}
	}
	if wantNumber {
		return parent + strconv.Itoa(top+1)
	}
	return parent + Letters(top+1)
}
