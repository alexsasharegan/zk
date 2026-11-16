package cli

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexsasharegan/zk/internal/config"
	"github.com/alexsasharegan/zk/internal/open"
	"github.com/alexsasharegan/zk/internal/tree"
	"github.com/alexsasharegan/zk/internal/tui"
)

// listBox is a box with four cards: 1, 1a (a claim), 1b and 2.
func listBox(t *testing.T) *boxEnv {
	t.Helper()
	b := newBox(t, defaultSpec)
	b.ok("new", "idea", "One")
	b.ok("new", "claim", "A", "--under", "1")
	b.ok("new", "idea", "B", "--under", "1")
	b.ok("new", "idea", "Two")
	return b
}

func listKeys(in tui.ListInput) string {
	var keys []string
	for _, c := range in.Cards {
		keys = append(keys, c.Key)
	}
	return strings.Join(keys, " ")
}

// On a terminal zk list prints nothing and opens the interactive list, with
// the lines zk list would print (ADR 261115-9pqw).
func TestListOpensTheInteractiveListOnATerminal(t *testing.T) {
	b := listBox(t)
	plain := b.ok("list")
	plainTree := strings.Split(b.ok("list", "--tree"), "\n")

	b.terminal = true
	if out := b.ok("list"); out != "" {
		t.Errorf("the interactive list prints nothing itself: %q", out)
	}
	if len(b.lists) != 1 {
		t.Fatalf("the interactive list ran %d times", len(b.lists))
	}
	in := b.lists[0]
	if in.Box != "Slip Box" || !strings.HasPrefix(in.Reason, "found walking up") || in.Root != "Slip Box" || in.Tree || in.Glyphs != tree.Unicode {
		t.Errorf("input: %+v", in)
	}
	var rows strings.Builder
	for i, c := range in.Cards {
		rows.WriteString(c.Row + "\n")
		if !strings.HasSuffix(plainTree[i+1], c.Branch) {
			t.Errorf("card %s reads %q in the tree, zk list --tree prints %q", c.Key, c.Branch, plainTree[i+1])
		}
	}
	if rows.String() != plain {
		t.Errorf("the rows are not what zk list prints:\n%s\n%s", rows.String(), plain)
	}
	if c := in.Cards[1]; c.Key != "1a" || c.Title != "A" || c.Kind != "claim" || c.Status != "raw" || c.Level != 2 {
		t.Errorf("card: %+v", c)
	}

	// The alias, and the flags.
	b.ok("ls", "--tree", "--ascii")
	if in := b.lists[1]; !in.Tree || in.Glyphs != tree.ASCII || len(in.Cards) != 4 {
		t.Errorf("--tree --ascii: %+v", in)
	}
	b.ok("list", "--kind", "claim")
	if got := listKeys(b.lists[2]); got != "1a" {
		t.Errorf("--kind: %q", got)
	}
	b.ok("list", "--status", "raw", "--under", "1")
	if in := b.lists[3]; listKeys(in) != "1a 1b" || in.Root != "1  One  (idea, raw)" {
		t.Errorf("--status --under: %q under %q", listKeys(in), in.Root)
	}
	b.ok("list", "--status", "linked")
	if in := b.lists[4]; len(in.Cards) != 0 {
		t.Errorf("no match still opens the list, empty: %+v", in.Cards)
	}
	if code, _, _ := b.zk("list", "--status", "done"); code != ExitUsage || len(b.lists) != 5 {
		t.Error("a bad --status is a usage error, and opens nothing")
	}

	b.list = func(tui.ListInput) error { return errors.New("no terminal after all") }
	if code, _, errOut := b.zk("list"); code != ExitFail || !strings.Contains(errOut, "no terminal after all") {
		t.Errorf("a failed list: %d %q", code, errOut)
	}
}

// With no terminal, --json, --no-input or ZK_NO_INPUT=1, zk list prints
// what it always printed, and no list opens.
func TestListPlainOutputIsUnchanged(t *testing.T) {
	b := listBox(t)
	want := map[string]string{
		"list":                  "1          raw       idea      One\n1a         raw       claim     A\n1b         raw       idea      B\n2          raw       idea      Two\n",
		"list tree":             "Slip Box\n├── 1  One  (idea, raw)\n│   ├── 1a  A  (claim, raw)\n│   └── 1b  B  (idea, raw)\n└── 2  Two  (idea, raw)\n\n4 cards\n",
		"list tree ascii under": "1  One  (idea, raw)\n|-- 1a  A  (claim, raw)\n`-- 1b  B  (idea, raw)\n\n2 cards\n",
		"list kind":             "1a         raw       claim     A\n",
	}
	args := map[string][]string{
		"list":                  {"list"},
		"list tree":             {"ls", "--tree"},
		"list tree ascii under": {"list", "--tree", "--ascii", "--under", "1"},
		"list kind":             {"list", "--kind", "claim"},
	}
	check := func(mode string, extra ...string) {
		t.Helper()
		for name, a := range args {
			if out := b.ok(append(append([]string{}, a...), extra...)...); out != want[name] {
				t.Errorf("%s, %s:\n%q\nwant\n%q", mode, name, out, want[name])
			}
		}
	}
	check("no terminal")
	b.terminal = true
	check("--no-input", "--no-input")
	b.vars["ZK_NO_INPUT"] = "1"
	check("ZK_NO_INPUT=1")
	delete(b.vars, "ZK_NO_INPUT")

	// --json is the same on a terminal and off it.
	b.terminal = false
	off := b.ok("list", "--json")
	offTree := b.ok("list", "--tree", "--json")
	b.terminal = true
	if on := b.ok("list", "--json"); on != off || strings.Count(on, "\n") != 4 || !strings.HasPrefix(on, `{"key":"1","level":1,"status":"raw","kind":"idea","title":"One","path":`) {
		t.Errorf("--json on a terminal:\n%s\noff it:\n%s", on, off)
	}
	if on := b.ok("list", "--tree", "--json"); on != offTree || on != off {
		t.Errorf("--tree --json on a terminal:\n%s", on)
	}
	if len(b.lists) != 0 {
		t.Errorf("the interactive list opened %d times", len(b.lists))
	}
}

// Enter opens the card as ADR 261115-mjrm says: Obsidian, or the editor
// with --open editor. The e key always has the editor.
func TestListOpensCards(t *testing.T) {
	b := listBox(t)
	b.terminal = true
	b.vars["EDITOR"] = "nvim"
	link := "obsidian obsidian://open?path=" + strings.ReplaceAll(url.QueryEscape(b.card("1a")), "+", "%20")

	b.list = func(in tui.ListInput) error {
		if in.Where != "Obsidian" || in.Attached || in.Editor != "nvim" {
			t.Errorf("by default: %q %v %q", in.Where, in.Attached, in.Editor)
		}
		if err := in.Open(in.Cards[1]); err != nil {
			t.Error(err)
		}
		return in.Edit(in.Cards[3])
	}
	b.ok("list")
	if len(b.opened) != 2 || b.opened[0] != link || b.opened[1] != "nvim "+b.card("2") {
		t.Errorf("opened: %v", b.opened)
	}

	b.opened = nil
	b.list = func(in tui.ListInput) error {
		if in.Where != "nvim" || !in.Attached {
			t.Errorf("--open editor: %q %v", in.Where, in.Attached)
		}
		return in.Open(in.Cards[1])
	}
	b.ok("list", "--open", "editor")
	if len(b.opened) != 1 || b.opened[0] != "nvim "+b.card("1a") {
		t.Errorf("--open editor: %v", b.opened)
	}

	// A failure goes back to the list, which says why.
	b.openErr = map[string]error{"obsidian": errors.New("no handler")}
	b.list = func(in tui.ListInput) error {
		if err := in.Open(in.Cards[0]); err == nil || !strings.Contains(err.Error(), "could not open it in Obsidian") {
			t.Errorf("a failed open: %v", err)
		}
		return nil
	}
	b.ok("list")

	b.list = nil
	n := len(b.lists)
	for _, bad := range []string{"none", "vim"} {
		if code, _, errOut := b.zk("list", "--open", bad); code != ExitUsage || !strings.Contains(errOut, "--open must be obsidian or editor") {
			t.Errorf("--open %s: %d %q", bad, code, errOut)
		}
	}
	if len(b.lists) != n {
		t.Error("a bad --open opens no list")
	}
}

// The header gets the box and why it was chosen, and the stderr line of
// ADR 261115-v1yz is printed after the list closes.
func TestListNoticeComesAfterTheListCloses(t *testing.T) {
	e := newNoticeEnv(t)
	elsewhere := filepath.Join(e.home, "code")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	var in tui.ListInput
	ran := false
	app := &App{Out: &out, Err: &errb,
		Env: func() (config.Env, error) {
			return config.Env{Getenv: func(k string) string {
				if k == "ZK_CONFIG" {
					return e.config
				}
				return ""
			}, Home: e.home, Cwd: elsewhere}, nil
		},
		Now:         time.Now,
		Terminal:    func() bool { return true },
		ErrTerminal: func() bool { return true },
		Opener:      open.Opener{GOOS: "darwin", Getenv: func(string) string { return "" }},
		ListUI: func(got tui.ListInput) error {
			in, ran = got, true
			if errb.Len() != 0 {
				t.Errorf("stderr before the list closed: %q", errb.String())
			}
			return nil
		},
	}
	if code := Run(app, []string{"list"}); code != ExitOK || !ran {
		t.Fatalf("exit %d, ran %v, stderr %q", code, ran, errb.String())
	}
	if in.Box != "Slip Box" || !strings.HasPrefix(in.Reason, "the default in ~") {
		t.Errorf("header: %q (%q)", in.Box, in.Reason)
	}
	if got, want := errb.String(), "zk: box notes at ~/vault/Slip Box (the default in ~/zk.toml.conf)\n"; got != want {
		t.Errorf("stderr after the list closed:\n%q\nwant\n%q", got, want)
	}
	if out.Len() != 0 {
		t.Errorf("stdout: %q", out.String())
	}
}
