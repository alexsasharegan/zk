package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// demoVault is the example vault in the repository.
const demoVault = "../../examples/demo-vault"

// copyDemo copies the demo vault into a temporary folder and returns a box
// environment rooted at its slip box.
func copyDemo(t *testing.T) *boxEnv {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	vault := filepath.Join(root, "vault")
	err = filepath.WalkDir(demoVault, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(demoVault, p)
		dst := filepath.Join(vault, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return &boxEnv{t: t, vault: vault, dir: filepath.Join(vault, "Slip Box"), vars: map[string]string{}}
}

// TestDemoVaultIsClean keeps the committed demo vault passing its own check.
func TestDemoVaultIsClean(t *testing.T) {
	b := copyDemo(t)
	if out := b.ok("check", "--strict"); out != "zk check: nothing found\n" {
		t.Fatalf("the demo vault has findings:\n%s", out)
	}
	if out := b.ok("fix"); out != "zk fix: nothing to fix\n" {
		t.Fatalf("zk fix would change the demo vault:\n%s", out)
	}
}

// TestParity runs the slip tool's test cases (scripts/test-slip.sh in the
// repository zk replaces), in their order, against a copy of the demo
// vault. The demo has roots 1 to 3, so new roots start at 4.
//
// Not ported, because zk has no such thing: the migration of a legacy
// `title:` field, the old `kind: research` source, the ./slip shim, and
// `new --help` listing the kinds (zk's kinds come from the box's spec, so
// zk lists them when a kind is missing or unknown, and zk config lists them).
func TestParity(t *testing.T) {
	b := copyDemo(t)
	step := func(name string, ok bool, detail ...any) {
		t.Helper()
		if !ok {
			t.Errorf("%s: %v", name, detail)
		}
	}
	has := func(s, sub string) bool { return strings.Contains(s, sub) }
	showField := func(k, field string) any {
		var info map[string]any
		json.Unmarshal([]byte(b.ok("show", k, "--json")), &info)
		return info[field]
	}
	expectFail := func(args ...string) string {
		t.Helper()
		code, out, errOut := b.zk(args...)
		if code != ExitFail {
			t.Errorf("zk %s: exit %d, want %d\n%s%s", strings.Join(args, " "), code, ExitFail, out, errOut)
		}
		return out + errOut
	}

	// Keys.
	step("sort of 7a1", showField("7a1", "sort") == "00000007/00000001/00000001")
	step("level of 7a1", showField("7a1", "level") == float64(3))
	step("sort of 7aa", showField("7aa", "sort") == "00000007/00000027")
	step("sort of 10", showField("10", "sort") == "00000010")

	// new makes cards in key order.
	for _, s := range []struct{ args, want string }{
		{"new idea One", "zk-4.md"},
		{"new idea A --under 4", "zk-4a.md"},
		{"new idea B --under 4", "zk-4b.md"},
		{"new question C --under 4a", "zk-4a1.md"},
		{"new idea D --under 4a1", "zk-4a1a.md"},
		{"new idea E", "zk-5.md"},
	} {
		out := b.ok(strings.Fields(s.args)...)
		step(s.args, out == "Slip Box/cards/"+s.want+"\n", out)
	}
	c4a1 := b.read(b.card("4a1"))
	step("new writes the tag", has(c4a1, "\n  - zk/4/a/1\n"))
	step("new writes the title and a Connections heading", has(c4a1, "\n# C\n") && has(c4a1, "\n## Connections\n"))
	step("new writes the nickname as the first alias, and no title field", has(c4a1, "aliases:\n  - C\n") && !has(c4a1, "zk_title"))
	step("raw cards keep the box clean", b.ok("check") == "zk check: nothing found\n")

	// The H1 is the title: editing it makes the alias stale, and fix repairs it.
	b.edit(b.card("4b"), func(s string) string { return strings.Replace(s, "# B\n", "# Bee\n", 1) })
	out := expectFail("check")
	step("a stale alias fails, naming aliases and the H1", has(out, "aliases[0]") && has(out, `"Bee"`), out)
	out = b.ok("fix")
	step("fix says what it changed", out == "fixed Slip Box/cards/zk-4b.md\n", out)
	step("fix rewrites the alias from the H1", has(b.read(b.card("4b")), "  - Bee\n"))

	// A nickname used twice warns, and fails under --strict.
	b.ok("retitle", "5", "Retitled")
	b.edit(b.card("5"), func(s string) string { return strings.Replace(s, "# Retitled\n", "# bread\n", 1) })
	b.ok("fix")
	code, out, _ := b.zk("check")
	step("a duplicate nickname warns but passes", code == ExitOK && has(out, "warning:") && has(out, "cards 1, 5"), code, out)
	expectFail("check", "--strict")
	b.edit(b.card("5"), func(s string) string { return strings.Replace(s, "# bread\n", "# E\n", 1) })
	b.ok("fix")
	step("the box is clean again", b.ok("check") == "zk check: nothing found\n")

	// A card made by hand is finished by fix (the file name is the key).
	write(t, b.card("6"), "---\nzk_kind: idea\nzk_status: raw\nzk_created: 2026-11-15T09:00:00\n---\n# Hand made\n")
	expectFail("check")
	out = b.ok("fix")
	step("fix names the hand-made card", has(out, "fixed Slip Box/cards/zk-6.md"), out)
	c6 := b.read(b.card("6"))
	step("fix writes key, level, sort, tag and alias", has(c6, `zk_key: "6"`) && has(c6, "zk_level: 1\n") && has(c6, `zk_sort: "00000006"`) && has(c6, "  - zk/6\n") && has(c6, "  - Hand made\n"), c6)
	step("fix is idempotent", b.ok("fix") == "zk fix: nothing to fix\n")

	// A skipped key warns, and fails under --strict.
	write(t, b.card("8"), "---\nzk_kind: idea\nzk_status: raw\nzk_created: 2026-11-15T09:00:00\n---\n# Gap\n")
	b.ok("fix")
	code, out, _ = b.zk("check")
	step("a skipped key warns but passes", code == ExitOK && has(out, "card 7 does not exist"), code, out)
	expectFail("check", "--strict")
	os.Remove(b.card("6"))
	os.Remove(b.card("8"))

	// new, list, show.
	step("a dry run names the card", b.ok("new", "idea", "Z", "--under", "4", "--dry-run") == "would make Slip Box/cards/zk-4c.md\n")
	_, err := os.Stat(b.card("4c"))
	step("a dry run writes nothing", os.IsNotExist(err))
	out = b.ok("new", "idea", "Two", "words", "here", "--under", "5", "--json")
	step("new --json names the key", has(out, `"key":"5a"`), out)
	step("title words are joined without quotes", has(b.read(b.card("5a")), "  - Two words here\n"))
	os.Remove(b.card("5a"))
	step("list --under lists the family", strings.Count(b.ok("list", "--under", "4"), "\n") == 4)
	step("list --json is one object per line", has(b.ok("list", "--kind", "question", "--json"), `"key":"4a1"`))
	step("list with no match prints nothing", b.ok("list", "--status", "linked", "--kind", "idea") == "")
	step("show gives the next child", showField("4a", "next_child") == "4a2")
	code, _, _ = b.zk("show", "0x")
	step("an invalid key is a usage error", code == ExitUsage)
	step("no arguments prints the command list", has(b.ok(), "Available Commands"))

	// A bare or wrong new names the kinds.
	code, _, errOut := b.zk("new")
	step("a bare new names the kinds", code == ExitUsage && has(errOut, "kinds in this box") && has(errOut, "lived through"), code, errOut)
	code, _, errOut = b.zk("new", "bogus", "Title")
	step("an unknown kind lists the choices", code == ExitUsage && has(errOut, `"bogus" is not a kind`) && has(errOut, "claim"), code, errOut)

	// check hints the fix, and marks fixable findings.
	b.edit(b.card("4b"), func(s string) string {
		return strings.Replace(s, `zk_sort: "00000004/00000002"`, `zk_sort: "wrong"`, 1)
	})
	out = expectFail("check")
	step("check hints the fix command", has(out, "run `zk fix 4b`"), out)
	_, out, _ = b.zk("check", "--json")
	step("a fixable finding is marked in json", has(out, `"fixable":true`), out)
	b.ok("fix")
	step("fix clears what check hinted", b.ok("check") == "zk check: nothing found\n")

	// retitle.
	out = b.ok("retitle", "5", "Fresh", "title")
	step("retitle reports the change", out == "retitled zk-5: \"E\" -> \"Fresh title\"\n", out)
	c5 := b.read(b.card("5"))
	step("retitle writes the H1 and the nickname", has(c5, "# Fresh title\n") && has(c5, "  - Fresh title\n"))
	code, _, _ = b.zk("retitle", "5", "bread")
	step("retitle refuses a nickname another card has, and changes nothing", code == ExitFail && has(b.read(b.card("5")), "# Fresh title\n"))
	b.edit(b.card("4"), func(s string) string { return s + "\nSee [[zk-5|(5) Fresh title]].\n" })
	out = b.ok("retitle", "5", "Newer", "title", "--links")
	step("retitle --links counts what it rewrote", has(out, "rewrote 1 link(s) in 1 file(s)"), out)
	step("retitle --links rewrites the old link text", has(b.read(b.card("4")), "(5) Newer title]]"))
	step("the box is clean after a retitle", b.ok("check") == "zk check: nothing found\n")

	// source new and done.
	out = b.ok("source", "new", "newsrc", "--provenance", "https://example.com", "--title", "New source")
	step("source new makes the page", out == "Slip Box/sources/newsrc.md\n", out)
	out = b.ok("source", "done", "newsrc", "4a", "4")
	step("source done lists the keys", out == "newsrc: extracted [4, 4a]\n", out)
	step("source done sets the status", has(b.read(filepath.Join(b.dir, "sources", "newsrc.md")), "zk_status: extracted\n"))
	code, _, _ = b.zk("source", "done", "newsrc", "99")
	step("source done refuses a card that does not exist", code == ExitUsage)
	step("a source made by zk passes the check", b.ok("check") == "zk check: nothing found\n")
	b.ok("source", "new", "filesrc", "--provenance", `podcast: a "show" #7 (box)`, "--form", "other")
	step("a provenance with a colon, a hash and quotes is quoted", has(b.read(filepath.Join(b.dir, "sources", "filesrc.md")), `zk_provenance: 'podcast: a "show" #7 (box)'`))

	// Derived fields.
	keep := b.read(b.card("4b"))
	b.edit(b.card("4b"), func(s string) string { return strings.Replace(s, "  - zk/4/b\n", "  - zk/4/c\n", 1) })
	step("a wrong tag fails", has(expectFail("check"), "tags should include zk/4/b"))
	b.edit(b.card("4b"), func(string) string { return keep })
	b.edit(b.card("4b"), func(s string) string {
		return strings.Replace(s, `zk_sort: "00000004/00000002"`, `zk_sort: "wrong"`, 1)
	})
	step("a wrong sort fails", has(expectFail("check"), "zk_sort should be"))
	b.edit(b.card("4b"), func(string) string { return keep })

	// A missing parent, and no reuse of a key after a gap.
	os.Remove(b.card("4a1a"))
	os.Remove(b.card("4a1"))
	os.Rename(b.card("4a"), b.card("4z"))
	b.ok("fix")
	b.ok("new", "idea", "Child", "--under", "4z")
	os.Remove(b.card("4z"))
	step("a missing parent fails", has(expectFail("check"), "parent card 4z does not exist"))
	os.Remove(b.card("4z1"))
	// 4a is gone and 4b remains, so the next child is 4c, not 4a again.
	step("a key is not reused after a gap", b.ok("new", "idea", "G", "--under", "4") == "Slip Box/cards/zk-4c.md\n")

	// The kinds' rules.
	for _, kind := range []string{"report", "quote"} {
		b.ok("new", kind, "K "+kind)
		step("a "+kind+" needs a source link", has(expectFail("check"), "must link to its source page"))
		os.Remove(b.card("6"))
	}
	b.ok("new", "hub", "H")
	step("a hub needs a link in Connections", has(expectFail("check"), "## Connections section with at least one link"))
	b.edit(b.card("6"), func(s string) string { return s + "\n[[zk-1]]: a hub this gathers.\n" })
	_, out, _ = b.zk("check")
	step("a hub needs no source", !has(out, "zk-6.md"), out)
	b.edit(b.card("6"), func(s string) string { return strings.Replace(s, `zk_key: "6"`, "zk_key: 6", 1) })
	step("an unquoted key is flagged", has(expectFail("check"), "quote the key"))
}

// TestDemoVaultIsWhatInitWrites keeps the demo box's spec, hub and base the
// files zk init writes (ADR 261115-j835).
func TestDemoVaultIsWhatInitWrites(t *testing.T) {
	b := copyDemo(t)
	fresh := filepath.Join(b.vault, "Fresh", "Slip Box")
	b.ok("init", fresh)
	for _, f := range []string{"zk.toml", "Slip Box Hub.md", "Slip Box Base.base"} {
		want, err := os.ReadFile(filepath.Join(demoVault, "Slip Box", f))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(fresh, f))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s differs from what zk init writes; regenerate the demo with zk init", f)
		}
	}
}
