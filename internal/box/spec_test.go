package box

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vault makes an Obsidian vault in a temp folder and writes a spec at rel
// (relative to the vault) with the given body. It returns the spec path.
func vault(t *testing.T, rel, body string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDefaults(t *testing.T) {
	path := vault(t, "Slip Box/zk.toml", "name = \"Slip Box\"\n")
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(path)
	if s.Dir != dir || s.Vault != filepath.Dir(dir) {
		t.Errorf("dir %q, vault %q", s.Dir, s.Vault)
	}
	if s.Cards != filepath.Join(dir, "cards") || s.Sources != filepath.Join(dir, "sources") {
		t.Errorf("cards %q, sources %q", s.Cards, s.Sources)
	}
	if s.Hub != "" || s.Base != "" {
		t.Errorf("hub %q, base %q: want none", s.Hub, s.Base)
	}
	if s.KeyPrefix != "zk" {
		t.Errorf("key prefix %q", s.KeyPrefix)
	}
	if got := strings.Join(s.KindNames(), " "); got != "claim fact hub idea question quote report" {
		t.Errorf("default kinds: %s", got)
	}
	if f := s.Mapping.Field("created"); f.Name != "zk_created" || f.Format != FormatDatetime {
		t.Errorf("default created field: %+v", f)
	}
	if v := s.Mapping.ClassValue(ClassCard); v != "card" {
		t.Errorf("default card class %q", v)
	}
}

// pkmLike is a spec shaped like a vault with its own field registry: the
// class in type, the card kind in kind, created_at as a date.
const pkmLike = `
name = "Slip Box"
hub = "../Slip Box Hub.md"
base = "../Slip Box Base.base"
cards = "cards"
sources = "sources"

[kinds.claim]
description = "Someone asserts this."
needs_source = true

[kinds.idea]
description = "Your own thought."

[classes]
card = "zk-card"
source = "zk-source"

[fields]
class = "type"
kind = "kind"
status = "status"
created = { name = "created_at", format = "date" }
title = "title"
author = "author"
published = { name = "published" }
authored_by = "authored_by"
series = "series"
`

func TestLoadMapped(t *testing.T) {
	path := vault(t, "3 AREAS/Slip Box/zk.toml", pkmLike)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Hub != filepath.Join(filepath.Dir(filepath.Dir(path)), "Slip Box Hub.md") {
		t.Errorf("hub %q", s.Hub)
	}
	if got := strings.Join(s.KindNames(), " "); got != "claim idea" {
		t.Errorf("kinds: %s (a spec's kinds replace the defaults)", got)
	}
	if !s.Kinds["claim"].NeedsSource || s.Kinds["idea"].NeedsSource {
		t.Errorf("kind rules: %+v", s.Kinds)
	}
	m := s.Mapping
	for canon, want := range map[string]Field{
		"class":     {Name: "type"},
		"created":   {Name: "created_at", Format: FormatDate},
		"published": {Name: "published", Format: FormatDate},
		"key":       {Name: "zk_key"},
		"sort":      {Name: "zk_sort"},
	} {
		if got := m.Field(canon); got != want {
			t.Errorf("%s: got %+v, want %+v", canon, got, want)
		}
	}
	if m.ClassValue(ClassSource) != "zk-source" {
		t.Errorf("source class %q", m.ClassValue(ClassSource))
	}
}

func TestLoadRejects(t *testing.T) {
	cases := map[string]string{
		"no name":                 `cards = "cards"`,
		"unknown key":             "name = \"x\"\ncolour = \"blue\"",
		"unknown key in a field":  "name = \"x\"\n[fields]\ncreated = { name = \"created_at\", bogus = 1 }",
		"path outside the vault":  "name = \"x\"\ncards = \"../../elsewhere\"",
		"absolute path":           "name = \"x\"\ncards = \"/tmp/cards\"",
		"cards same as sources":   "name = \"x\"\ncards = \"notes\"\nsources = \"notes\"",
		"bad key prefix":          "name = \"x\"\nkey_prefix = \"Z K\"",
		"bad kind name":           "name = \"x\"\n[kinds.Claim]\ndescription = \"d\"",
		"kind without text":       "name = \"x\"\n[kinds.claim]\nneeds_source = true",
		"unknown canonical field": "name = \"x\"\n[fields]\ncolour = \"colour\"",
		"map onto tags":           "name = \"x\"\n[fields]\nkind = \"tags\"",
		"map onto aliases":        "name = \"x\"\n[fields]\ntitle = \"aliases\"",
		"two fields, one name":    "name = \"x\"\n[fields]\nkind = \"type\"\nclass = \"type\"",
		"clash with a default":    "name = \"x\"\n[fields]\nkind = \"zk_key\"",
		"format on a non-date":    "name = \"x\"\n[fields]\nkind = { name = \"kind\", format = \"date\" }",
		"bad format":              "name = \"x\"\n[fields]\ncreated = { name = \"c\", format = \"epoch\" }",
		"field not text or table": "name = \"x\"\n[fields]\nkind = 3",
		"unknown class":           "name = \"x\"\n[classes]\ntrail = \"zk-trail\"",
		"same class values":       "name = \"x\"\n[classes]\ncard = \"note\"\nsource = \"note\"",
		"bad toml":                "name = ",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(vault(t, "box/zk.toml", body)); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestLoadOutsideVault(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "zk.toml")
	if err := os.WriteFile(p, []byte("name = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Load(p)
	if err == nil || !strings.Contains(err.Error(), "not inside an Obsidian vault") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadMissingHint(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "zk.toml"))
	if err == nil || !strings.Contains(err.Error(), "Obsidian Sync") {
		t.Fatalf("got %v, want the sync hint", err)
	}
}
