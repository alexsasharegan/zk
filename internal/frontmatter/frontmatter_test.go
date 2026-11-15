package frontmatter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexsasharegan/zk/internal/box"
	"github.com/alexsasharegan/zk/internal/frontmatter"
)

// card is a card in the shape Obsidian and the slip tool write, with a
// horizontal rule in the body that must not be taken for a fence.
const card = `---
key: "2a"
level: 2
sort: "00000002/00000001"
kind: card
type: claim
aliases:
  - "Grassé's coinage"
status: linked
created_at: "2026-11-13T00:16:01"
tags:
  - zk/2/a
# a comment Alexander left
---
# Grassé's coinage

Pierre-Paul Grassé coined *stigmergy* in 1959.

---

## Connections
`

func parse(t *testing.T, s string) *frontmatter.Doc {
	t.Helper()
	d, err := frontmatter.Parse([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func bytesOf(t *testing.T, d *frontmatter.Doc) string {
	t.Helper()
	b, err := d.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUnchangedNoteIsByteForByte(t *testing.T) {
	d := parse(t, card)
	if got := bytesOf(t, d); got != card {
		t.Fatalf("round trip changed the note:\n%s", got)
	}
	if err := d.Update(func(map[string]any) {}); err != nil {
		t.Fatal(err)
	}
	if got := bytesOf(t, d); got != card {
		t.Fatalf("a no-op Update changed the note:\n%s", got)
	}
	if !strings.HasPrefix(string(d.Body()), "# Grassé's coinage") {
		t.Errorf("body starts %.30q", d.Body())
	}
}

func TestUpdateKeepsOrderQuotingAndBody(t *testing.T) {
	d := parse(t, card)
	err := d.Update(func(m map[string]any) {
		m["status"] = "proposed"   // changed in place
		m["key"] = "2b"            // was quoted, stays quoted
		delete(m, "level")         // removed
		m["zk_note"] = "added"     // new, at the end
		m["authored_by"] = "agent" // new, sorted before zk_note
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `---
key: "2b"
sort: "00000002/00000001"
kind: card
type: claim
aliases:
  - "Grassé's coinage"
status: proposed
created_at: "2026-11-13T00:16:01"
tags:
  - zk/2/a
# a comment Alexander left

authored_by: agent
zk_note: added
---
`
	// The YAML library puts a blank line after a comment that ends a block
	// when keys follow it. It is valid YAML and Obsidian ignores it.
	got := bytesOf(t, d)
	if !strings.HasPrefix(got, want) {
		t.Fatalf("got:\n%s\nwant prefix:\n%s", got, want)
	}
	if !strings.HasSuffix(got, "\n## Connections\n") || !strings.Contains(got, "\n---\n\n## Connections") {
		t.Errorf("body changed:\n%s", got)
	}
}

func TestNoFrontMatter(t *testing.T) {
	const note = "# Just a note\n\nNo fields.\n"
	d := parse(t, note)
	if got := bytesOf(t, d); got != note {
		t.Fatalf("got %q", got)
	}
	if err := d.Update(func(m map[string]any) { m["status"] = "raw" }); err != nil {
		t.Fatal(err)
	}
	if got := bytesOf(t, d); got != "---\nstatus: raw\n---\n"+note {
		t.Fatalf("got %q", got)
	}
}

func TestEmptyAndUnclosedBlocks(t *testing.T) {
	d := parse(t, "---\n---\nbody\n")
	if fm, _ := d.Fields(); len(fm) != 0 || string(d.Body()) != "body\n" {
		t.Errorf("empty block: %v, body %q", fm, d.Body())
	}
	if got := bytesOf(t, d); got != "---\n---\nbody\n" {
		t.Errorf("empty block round trip: %q", got)
	}

	const unclosed = "---\nnot: closed\n"
	d = parse(t, unclosed)
	if fm, _ := d.Fields(); len(fm) != 0 || string(d.Body()) != unclosed {
		t.Errorf("an unclosed fence is body text: %v, body %q", fm, d.Body())
	}

	d = parse(t, "---\nkey: v\n---")
	if fm, _ := d.Fields(); fm["key"] != "v" || len(d.Body()) != 0 {
		t.Errorf("closing fence at end of file: %v, body %q", fm, d.Body())
	}
}

func TestRejectsBadFrontMatter(t *testing.T) {
	for _, s := range []string{"---\n- a\n- b\n---\n", "---\nkey: [unclosed\n---\n", "---\njust text\n---\n"} {
		if _, err := frontmatter.Parse([]byte(s)); err == nil {
			t.Errorf("%q: want an error", s)
		}
	}
}

// TestBoxCardInFile writes a card through a mapping like a vault with its
// own field registry, then reads it back from the file.
func TestBoxCardInFile(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(root, "box", "zk.toml")
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := "name = \"Box\"\n[classes]\ncard = \"zk-card\"\n[fields]\nclass = \"type\"\nkind = \"kind\"\nstatus = \"status\"\ncreated = { name = \"created_at\", format = \"date\" }\n"
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := box.Load(specPath)
	if err != nil {
		t.Fatal(err)
	}

	c := box.Card{Kind: "claim", Key: "7", Level: 1, Sort: "00000007", Status: "raw", Created: time.Date(2026, 11, 15, 0, 0, 0, 0, time.Local)}
	d := parse(t, "# Title\n")
	if err := d.Update(func(m map[string]any) {
		s.Mapping.PutCard(m, c)
		m["aliases"] = []string{"Title"}
		m["tags"] = []string{"zk/7"}
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "box", "cards", "zk-7.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := frontmatter.WriteFile(path, d); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	want := `---
aliases:
  - Title
created_at: 2026-11-15
kind: claim
status: raw
tags:
  - zk/7
type: zk-card
zk_key: "7"
zk_level: 1
zk_sort: "00000007"
---
# Title
`
	if string(raw) != want {
		t.Fatalf("file:\n%s\nwant:\n%s", raw, want)
	}

	back, err := frontmatter.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm, err := back.Fields()
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Mapping.Card(fm)
	if err != nil || got != c {
		t.Fatalf("read back %+v, %v; want %+v", got, err, c)
	}
}

func TestWriteFileKeepsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(path, []byte("body\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := parse(t, "body\n")
	if err := frontmatter.WriteFile(path, d); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", info.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("temporary file left behind: %v", entries)
	}
}
