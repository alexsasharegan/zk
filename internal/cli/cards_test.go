package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/alexsasharegan/zk/internal/config"
)

// boxEnv is a vault with a box in it, and a way to run zk inside it.
type boxEnv struct {
	t        *testing.T
	vault    string
	dir      string // the box folder
	terminal bool
	vars     map[string]string
}

func newBox(t *testing.T, spec string) *boxEnv {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b := &boxEnv{t: t, vault: filepath.Join(root, "vault"), vars: map[string]string{}}
	b.dir = filepath.Join(b.vault, "Slip Box")
	write(t, filepath.Join(b.vault, ".obsidian", "app.json"), "{}")
	write(t, filepath.Join(b.dir, "zk.toml"), spec)
	return b
}

const defaultSpec = "name = \"Slip Box\"\n"

func (b *boxEnv) zk(args ...string) (int, string, string) {
	b.t.Helper()
	var out, errb bytes.Buffer
	app := &App{
		Out: &out, Err: &errb,
		Env: func() (config.Env, error) {
			return config.Env{Getenv: func(k string) string { return b.vars[k] }, Home: filepath.Dir(b.vault), Cwd: b.dir}, nil
		},
		Now:      func() time.Time { return time.Date(2026, 11, 15, 9, 30, 0, 0, time.Local) },
		Terminal: func() bool { return b.terminal },
	}
	code := Run(app, args)
	return code, out.String(), errb.String()
}

// ok runs zk and fails the test unless it exits 0.
func (b *boxEnv) ok(args ...string) string {
	b.t.Helper()
	code, out, errOut := b.zk(args...)
	if code != ExitOK {
		b.t.Fatalf("zk %s: exit %d\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), code, out, errOut)
	}
	return out
}

func (b *boxEnv) card(k string) string { return filepath.Join(b.dir, "cards", "zk-"+k+".md") }

func (b *boxEnv) read(p string) string {
	b.t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		b.t.Fatal(err)
	}
	return string(data)
}

func (b *boxEnv) edit(p string, fn func(string) string) {
	b.t.Helper()
	if err := os.WriteFile(p, []byte(fn(b.read(p))), 0o644); err != nil {
		b.t.Fatal(err)
	}
}

func (b *boxEnv) source(slug, extra string) {
	write(b.t, filepath.Join(b.dir, "sources", slug+".md"),
		"---\nzk_class: source\nzk_title: S\nzk_provenance: https://example.com\nzk_form: article\nzk_status: read\nzk_created: 2026-01-01T00:00:00\n"+extra+"---\n# S\n")
}

func TestNewMakesCardsInKeyOrder(t *testing.T) {
	b := newBox(t, defaultSpec)
	steps := []struct {
		args []string
		want string
	}{
		{[]string{"new", "idea", "One"}, "zk-1.md"},
		{[]string{"new", "idea", "Two"}, "zk-2.md"},
		{[]string{"new", "claim", "A", "--under", "2"}, "zk-2a.md"},
		{[]string{"new", "idea", "B", "--under", "2"}, "zk-2b.md"},
		{[]string{"new", "question", "C", "--under", "2a"}, "zk-2a1.md"},
		{[]string{"new", "idea", "D", "--under", "2a1"}, "zk-2a1a.md"},
		{[]string{"new", "idea", "E"}, "zk-3.md"},
	}
	for _, s := range steps {
		if got := strings.TrimSpace(b.ok(s.args...)); got != "Slip Box/cards/"+s.want {
			t.Fatalf("zk %v: got %q, want %q", s.args, got, s.want)
		}
	}
	want := `---
zk_class: card
zk_kind: question
zk_key: "2a1"
zk_level: 3
zk_sort: "00000002/00000001/00000001"
aliases:
  - C
zk_status: raw
zk_created: 2026-11-15T09:30:00
tags:
  - zk/2/a/1
---
# C


## Connections
`
	if got := b.read(b.card("2a1")); got != want {
		t.Errorf("card file:\n%s\nwant:\n%s", got, want)
	}
}

func TestNewOptions(t *testing.T) {
	b := newBox(t, defaultSpec)
	b.ok("new", "idea", "One")
	if got := b.ok("new", "idea", "Z", "--under", "1", "--dry-run"); got != "would make Slip Box/cards/zk-1a.md\n" {
		t.Errorf("dry run: %q", got)
	}
	if _, err := os.Stat(b.card("1a")); err == nil {
		t.Error("a dry run wrote the card")
	}
	out := b.ok("new", "idea", "Two", "words", "here", "--under", "1", "--json", "--agent")
	var res map[string]any
	if json.Unmarshal([]byte(out), &res) != nil || res["key"] != "1a" {
		t.Fatalf("--json: %q", out)
	}
	text := b.read(b.card("1a"))
	if !strings.Contains(text, "  - Two words here\n") || !strings.Contains(text, "zk_authored_by: agent\n") {
		t.Errorf("title words and --agent:\n%s", text)
	}
	if code, _, errOut := b.zk("new", "idea", "X", "--under", "9"); code != ExitUsage || !strings.Contains(errOut, `no card "9"`) {
		t.Errorf("--under a missing card: %d %q", code, errOut)
	}
	if code, _, errOut := b.zk("new", "bogus", "Title"); code != ExitUsage || !strings.Contains(errOut, "claim") {
		t.Errorf("unknown kind: %d %q", code, errOut)
	}
}

// ADR 261115-czb2: with arguments missing and no terminal, zk exits 2 and
// never waits for input; on a terminal --no-input and ZK_NO_INPUT do the same.
func TestNewMissingArgumentsNeverPrompt(t *testing.T) {
	b := newBox(t, defaultSpec)
	for _, args := range [][]string{{"new"}, {"new", "idea"}} {
		code, _, errOut := b.zk(args...)
		if code != ExitUsage || !strings.Contains(errOut, "kinds in this box") || !strings.Contains(errOut, "first-hand") && !strings.Contains(errOut, "lived through") {
			t.Errorf("%v: exit %d, stderr %q", args, code, errOut)
		}
	}
	b.terminal = true
	if !strings.Contains(func() string { _, _, e := b.zk("new"); return e }(), "terminal UI") {
		t.Error("on a terminal the message should say the UI is not built yet")
	}
	for _, setup := range []func(){
		func() { b.vars["ZK_NO_INPUT"] = "1" },
	} {
		setup()
		if _, _, e := b.zk("new"); strings.Contains(e, "terminal UI") {
			t.Error("ZK_NO_INPUT=1 should behave as no terminal")
		}
	}
	delete(b.vars, "ZK_NO_INPUT")
	if _, _, e := b.zk("new", "--no-input"); strings.Contains(e, "terminal UI") {
		t.Error("--no-input should behave as no terminal")
	}
}

// linkUp makes the cards proposed and links them, so the check is clean.
func linkUp(b *boxEnv, keys ...string) {
	var all []string
	for _, k := range keys {
		all = append(all, "[[zk-"+k+"]]")
	}
	for i, k := range keys {
		b.edit(b.card(k), func(s string) string {
			s = strings.Replace(s, "zk_status: raw", "zk_status: proposed", 1)
			if i == 0 {
				return s + "\nSee " + strings.Join(all, " ") + ".\n"
			}
			return s + "\nSee [[zk-" + keys[0] + "]].\n"
		})
	}
}

func TestCheckCleanBoxAndFindings(t *testing.T) {
	b := newBox(t, defaultSpec)
	b.source("src", "")
	b.ok("new", "idea", "One")
	b.ok("new", "claim", "A", "--under", "1")
	linkUp(b, "1", "1a")
	b.edit(b.card("1a"), func(s string) string { return s + "Source: [[src]].\n" })
	if out := b.ok("check"); out != "zk check: nothing found\n" {
		t.Fatalf("a clean box: %q", out)
	}

	// A stale alias fails; fix repairs it from the H1.
	b.edit(b.card("1a"), func(s string) string { return strings.Replace(s, "# A\n", "# Ay\n", 1) })
	code, out, _ := b.zk("check")
	if code != ExitFail || !strings.Contains(out, "aliases[0]") || !strings.Contains(out, "run `zk fix 1a`") {
		t.Fatalf("stale alias: %d\n%s", code, out)
	}
	if out := b.ok("fix"); out != "fixed Slip Box/cards/zk-1a.md\n" {
		t.Errorf("fix: %q", out)
	}
	if !strings.Contains(b.read(b.card("1a")), "  - Ay\n") {
		t.Error("fix did not rewrite the alias")
	}
	if out := b.ok("fix"); out != "zk fix: nothing to fix\n" {
		t.Errorf("fix is not idempotent: %q", out)
	}

	// Derived fields: a wrong sort, an unquoted key, a missing tag.
	b.edit(b.card("1a"), func(s string) string {
		s = strings.Replace(s, `zk_sort: "00000001/00000001"`, `zk_sort: "wrong"`, 1)
		s = strings.Replace(s, `zk_key: "1a"`, `zk_key: 1a`, 1)
		return strings.Replace(s, "  - zk/1/a\n", "  - zk/1/b\n", 1)
	})
	_, out, _ = b.zk("check", "--json")
	for _, want := range []string{"zk_sort should be", "tags should include zk/1/a", `"fixable":true`} {
		if !strings.Contains(out, want) {
			t.Errorf("check --json lacks %q:\n%s", want, out)
		}
	}
	if out := b.ok("check", "--fix"); !strings.Contains(out, "fixed Slip Box/cards/zk-1a.md") || !strings.Contains(out, "nothing found") {
		t.Errorf("check --fix: %q", out)
	}
}

func TestCheckRules(t *testing.T) {
	b := newBox(t, defaultSpec)
	b.ok("new", "idea", "One")
	b.ok("new", "idea", "Two", "--under", "1")
	linkUp(b, "1", "1a")
	if out := b.ok("check"); !strings.Contains(out, "nothing found") {
		t.Fatalf("setup: %s", out)
	}
	expectFinding := func(name, want string) {
		t.Helper()
		code, out, _ := b.zk("check")
		if code != ExitFail || !strings.Contains(out, want) {
			t.Errorf("%s: exit %d, want %q in:\n%s", name, code, want, out)
		}
	}

	// A report, a quote and a claim need a source link; a hub needs a link in Connections.
	for _, kind := range []string{"report", "quote", "claim"} {
		b.ok("new", kind, "K "+kind)
		expectFinding(kind+" needs a source", "must link to its source page")
		os.Remove(b.card("2"))
	}
	b.ok("new", "hub", "H")
	expectFinding("hub needs a link", "## Connections section with at least one link")
	b.edit(b.card("2"), func(s string) string { return s + "[[zk-1]]: a person this hub gathers.\n" })
	if _, out, _ := b.zk("check"); strings.Contains(out, "zk-2.md [hub]") || strings.Contains(out, "zk-2.md [source]") {
		t.Errorf("a hub with a link needs nothing more:\n%s", out)
	}
	os.Remove(b.card("2"))

	// A missing parent, a broken link, an orphan.
	b.ok("new", "idea", "Grandchild", "--under", "1a")
	os.Rename(b.card("1a1"), b.card("1b1"))
	expectFinding("missing parent", "parent card 1b does not exist")
	os.Remove(b.card("1b1"))
	b.edit(b.card("1"), func(s string) string { return s + "[[nowhere]]\n" })
	expectFinding("broken link", "[[nowhere]] does not resolve")
	b.edit(b.card("1"), func(s string) string { return strings.Replace(s, "[[nowhere]]\n", "", 1) })
	b.ok("new", "idea", "Alone")
	b.edit(b.card("2"), func(s string) string { return strings.Replace(s, "zk_status: raw", "zk_status: proposed", 1) })
	expectFinding("orphan", "no card or source links to this card")
	os.Remove(b.card("2"))

	// A duplicate nickname and a skipped key warn, and fail under --strict.
	b.ok("new", "idea", "one")
	b.edit(b.card("1a"), func(s string) string { return s + "[[zk-2]]\n" })
	b.edit(b.card("2"), func(s string) string { return strings.Replace(s, "zk_status: raw", "zk_status: proposed", 1) })
	if code, out, _ := b.zk("check"); code != ExitOK || !strings.Contains(out, "warning:") || !strings.Contains(out, "cards 1, 2") {
		t.Errorf("duplicate nickname warns: %d\n%s", code, out)
	}
	if code, _, _ := b.zk("check", "--strict"); code != ExitFail {
		t.Error("duplicate nickname fails under --strict")
	}
	b.ok("retitle", "2", "Second")
	os.Rename(b.card("2"), b.card("3"))
	b.ok("fix")
	b.edit(b.card("1a"), func(s string) string { return strings.Replace(s, "[[zk-2]]", "[[zk-3]]", 1) })
	if code, out, _ := b.zk("check"); code != ExitOK || !strings.Contains(out, "card 2 does not exist") {
		t.Errorf("skipped key warns: %d\n%s", code, out)
	}
	if code, _, _ := b.zk("check", "--strict"); code != ExitFail {
		t.Error("skipped key fails under --strict")
	}
}

func TestFixFinishesAHandMadeCard(t *testing.T) {
	b := newBox(t, defaultSpec)
	write(t, b.card("4"), "---\nzk_kind: idea\nzk_status: raw\nzk_created: 2026-11-15T09:00:00\n---\n# Hand made\n")
	if code, _, _ := b.zk("check"); code != ExitFail {
		t.Error("the bare card should fail the check")
	}
	if out := b.ok("fix", "4"); out != "fixed Slip Box/cards/zk-4.md\n" {
		t.Errorf("fix: %q", out)
	}
	text := b.read(b.card("4"))
	for _, want := range []string{"zk_class: card\n", `zk_key: "4"`, "zk_level: 1\n", `zk_sort: "00000004"`, "  - zk/4\n", "  - Hand made\n"} {
		if !strings.Contains(text, want) {
			t.Errorf("fixed card lacks %q:\n%s", want, text)
		}
	}
	if code, _, errOut := b.zk("fix", "9"); code != ExitUsage || !strings.Contains(errOut, `no card "9"`) {
		t.Errorf("fix of a missing card: %d %q", code, errOut)
	}
}

func TestListAndShow(t *testing.T) {
	b := newBox(t, defaultSpec)
	b.source("src", "zk_extracted:\n  - \"1a\"\n")
	b.ok("new", "idea", "One")
	b.ok("new", "claim", "A", "--under", "1")
	b.ok("new", "idea", "B", "--under", "1")
	b.ok("new", "idea", "Two")
	b.edit(b.card("1"), func(s string) string { return s + "See [[zk-1a]] and [[src]].\n" })

	if out := b.ok("list", "--under", "1"); strings.Count(out, "\n") != 2 {
		t.Errorf("list --under: %q", out)
	}
	if out := b.ok("ls", "--tree"); !regexp.MustCompile(`(?m)^  1a\s+raw\s+claim\s+A$`).MatchString(out) {
		t.Errorf("ls --tree:\n%s", out)
	}
	out := b.ok("list", "--kind", "claim", "--json")
	var row map[string]any
	if json.Unmarshal([]byte(out), &row) != nil || row["key"] != "1a" || row["title"] != "A" {
		t.Errorf("list --json: %q", out)
	}
	if out := b.ok("list", "--status", "linked"); out != "" {
		t.Errorf("no match prints nothing: %q", out)
	}
	if code, _, _ := b.zk("list", "--status", "done"); code != ExitUsage {
		t.Error("a bad --status is a usage error")
	}

	out = b.ok("show", "1a")
	for _, want := range []string{"next_child     1a1", "in   zk-1", "in   src", "extracted"} {
		if !strings.Contains(strings.Join(strings.Fields(out), " "), strings.Join(strings.Fields(want), " ")) {
			t.Errorf("show lacks %q:\n%s", want, out)
		}
	}
	out = b.ok("show", "1", "--json")
	var info struct {
		Links []struct{ Direction, Target, Kind string }
	}
	if json.Unmarshal([]byte(out), &info) != nil || len(info.Links) != 2 || info.Links[1].Kind != "source" {
		t.Errorf("show --json: %s", out)
	}
	if out := b.ok("show", "7b"); !strings.Contains(out, "exists         false") {
		t.Errorf("show of a free key: %s", out)
	}
	if code, _, _ := b.zk("show", "0x"); code != ExitUsage {
		t.Error("an invalid key is a usage error")
	}
}

func TestRetitle(t *testing.T) {
	b := newBox(t, defaultSpec)
	b.ok("new", "idea", "One")
	b.ok("new", "idea", "E")
	linkUp(b, "1", "2")
	if out := b.ok("retitle", "2", "Fresh", "title"); out != "retitled zk-2: \"E\" -> \"Fresh title\"\n" {
		t.Errorf("retitle: %q", out)
	}
	text := b.read(b.card("2"))
	if !strings.Contains(text, "# Fresh title\n") || !strings.Contains(text, "  - Fresh title\n") {
		t.Errorf("retitle writes the H1 and the nickname:\n%s", text)
	}
	if code, _, _ := b.zk("retitle", "2", "one"); code != ExitFail || !strings.Contains(b.read(b.card("2")), "# Fresh title") {
		t.Error("retitle refuses another card's nickname and changes nothing")
	}
	b.edit(b.card("1"), func(s string) string { return s + "See [[zk-2|(2) Fresh title]].\n" })
	if out := b.ok("retitle", "2", "Newer", "--links"); !strings.Contains(out, "rewrote 1 link(s) in 1 file(s)") {
		t.Errorf("retitle --links: %q", out)
	}
	if !strings.Contains(b.read(b.card("1")), "(2) Newer]]") {
		t.Error("retitle --links did not rewrite the link")
	}
	if out := b.ok("check"); !strings.Contains(out, "nothing found") {
		t.Errorf("the box is clean after a retitle:\n%s", out)
	}
}

func TestSources(t *testing.T) {
	b := newBox(t, defaultSpec)
	b.ok("new", "idea", "One")
	b.ok("new", "claim", "A", "--under", "1")
	linkUp(b, "1", "1a")
	if out := b.ok("source", "new", "newsrc", "--provenance", "https://example.com", "--title", "New source"); out != "Slip Box/sources/newsrc.md\n" {
		t.Errorf("source new: %q", out)
	}
	b.edit(b.card("1a"), func(s string) string { return s + "[^1]: [[newsrc]]\n" })
	if out := b.ok("source", "done", "newsrc", "1a", "1"); out != "newsrc: extracted [1, 1a]\n" {
		t.Errorf("source done: %q", out)
	}
	text := b.read(filepath.Join(b.dir, "sources", "newsrc.md"))
	if !strings.Contains(text, "zk_status: extracted\n") || !strings.Contains(text, "zk_extracted:\n  - \"1\"\n  - \"1a\"\n") {
		t.Errorf("source page:\n%s", text)
	}
	if code, _, _ := b.zk("source", "done", "newsrc", "99"); code != ExitUsage {
		t.Error("source done refuses a card that does not exist")
	}
	if out := b.ok("check"); !strings.Contains(out, "nothing found") {
		t.Errorf("a source made by zk passes the check:\n%s", out)
	}

	b.ok("source", "new", "box", "--provenance", `podcast: a "show" #7 (box)`, "--form", "other")
	if text := b.read(filepath.Join(b.dir, "sources", "box.md")); !strings.Contains(text, `zk_provenance: 'podcast: a "show" #7 (box)'`) {
		t.Errorf("a provenance with a colon, a hash and quotes is quoted:\n%s", text)
	}
	if code, _, _ := b.zk("source", "new", "Bad Slug", "--provenance", "x"); code != ExitUsage {
		t.Error("a bad slug is a usage error")
	}
	if code, _, _ := b.zk("source", "new", "noprov"); code != ExitUsage {
		t.Error("a source needs --provenance")
	}
	if code, _, _ := b.zk("source", "new", "box", "--provenance", "x"); code != ExitFail {
		t.Error("an existing source is a failure")
	}
}

// A box whose spec maps onto a vault's own fields writes those names.
func TestMappedBoxWritesVaultFields(t *testing.T) {
	b := newBox(t, `name = "Slip Box"
[classes]
card = "zk-card"
source = "zk-source"
[fields]
class = "type"
kind = "kind"
status = "status"
created = { name = "created_at", format = "date" }
`)
	b.ok("new", "idea", "One")
	text := b.read(b.card("1"))
	for _, want := range []string{"type: zk-card\n", "kind: idea\n", "status: raw\n", "created_at: 2026-11-15\n", `zk_key: "1"`} {
		if !strings.Contains(text, want) {
			t.Errorf("lacks %q:\n%s", want, text)
		}
	}
	if code, out, _ := b.zk("check"); code != ExitOK {
		t.Errorf("a raw card in a mapped box passes: %d\n%s", code, out)
	}
}
