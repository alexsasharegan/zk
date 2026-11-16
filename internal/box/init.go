package box

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SpecFile is the name of a box's spec.
const SpecFile = "zk.toml"

// InitResult is what Init made.
type InitResult struct {
	Spec      *Spec
	Wrote     []string // files written, absolute
	Made      []string // folders created, absolute
	Types     []string // property types added to .obsidian/types.json
	Conflicts []TypeConflict
}

// badNameChars are the characters Obsidian cannot use in a file name or a
// link.
const badNameChars = `/\:*?"<>|#^[]`

// Init makes a box in dir, named name (the folder's name when empty): the
// spec, the hub note, the base, and the cards and sources folders, then the
// missing property types (ADR 261115-j835). It checks everything before it
// writes anything, and never overwrites a file.
func Init(dir, name string) (*InitResult, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = filepath.Base(dir)
	}
	if err := checkBoxName(name); err != nil {
		return nil, err
	}

	existing, err := nearestExisting(dir)
	if err != nil {
		return nil, err
	}
	vault, err := findVault(existing)
	if err != nil {
		return nil, err
	}
	for p := dir; ; p = filepath.Dir(p) {
		if _, err := os.Stat(filepath.Join(p, SpecFile)); err == nil {
			return nil, fmt.Errorf("%s is already a box (its spec is %s); a box cannot hold another", p, filepath.Join(p, SpecFile))
		}
		if p == vault || filepath.Dir(p) == p {
			break
		}
	}
	if inner, err := specBelow(dir); err != nil {
		return nil, err
	} else if inner != "" {
		return nil, fmt.Errorf("%s holds a box already (its spec is %s); a box cannot hold another", dir, inner)
	}

	m := DefaultMapping()
	files := []struct{ name, body string }{
		{SpecFile, initSpec(name)},
		{name + " Hub.md", initHub(name)},
		{name + " Base.base", initBase(m)},
	}
	var taken []string
	for _, f := range files {
		if _, err := os.Lstat(filepath.Join(dir, f.name)); err == nil {
			taken = append(taken, f.name)
		}
	}
	if len(taken) > 0 {
		return nil, fmt.Errorf("%s already holds %s; zk init never overwrites a file", dir, strings.Join(taken, ", "))
	}
	for _, folder := range []string{"cards", "sources"} {
		if info, err := os.Stat(filepath.Join(dir, folder)); err == nil && !info.IsDir() {
			return nil, fmt.Errorf("%s is a file, and the box needs a folder there", filepath.Join(dir, folder))
		}
	}

	res := &InitResult{}
	for _, d := range []string{dir, filepath.Join(dir, "cards"), filepath.Join(dir, "sources")} {
		if _, err := os.Stat(d); errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(d, 0o755); err != nil {
				return res, err
			}
			res.Made = append(res.Made, d)
		}
	}
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		// O_EXCL: a file that appeared since the check is still not replaced.
		fh, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return res, err
		}
		_, werr := fh.WriteString(f.body)
		if cerr := fh.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return res, werr
		}
		res.Wrote = append(res.Wrote, p)
	}

	spec, err := Load(filepath.Join(dir, SpecFile))
	if err != nil {
		return res, fmt.Errorf("zk init wrote a spec it cannot read (a bug): %w", err)
	}
	res.Spec = spec
	res.Types, res.Conflicts, err = AddTypes(spec.Vault, spec.Mapping.PropertyTypes())
	return res, err
}

// specBelow returns a box spec inside dir, or "" when there is none or dir
// does not exist. Dot folders (.obsidian, .git, .trash) are skipped.
func specBelow(dir string) (string, error) {
	found := ""
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == dir && errors.Is(err, fs.ErrNotExist) {
				return filepath.SkipAll
			}
			return err
		}
		if d.IsDir() && p != dir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == SpecFile {
			found = p
			return filepath.SkipAll
		}
		return nil
	})
	return found, err
}

func checkBoxName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return argErr("a box needs a name")
	case strings.HasPrefix(name, "."):
		return argErr("the box name %q starts with a dot, which hides its files", name)
	case strings.ContainsAny(name, badNameChars):
		return argErr("the box name %q holds a character Obsidian cannot use in a file name or link (%s)", name, strings.Join(strings.Split(badNameChars, ""), " "))
	case name != strings.TrimSpace(name):
		return argErr("the box name %q starts or ends with a space", name)
	}
	return nil
}

// nearestExisting returns dir, or its nearest ancestor that exists. It must
// be a folder.
func nearestExisting(dir string) (string, error) {
	for p := dir; ; p = filepath.Dir(p) {
		info, err := os.Stat(p)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s is a file, not a folder", p)
			}
			return p, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if filepath.Dir(p) == p {
			return "", fmt.Errorf("no folder of %s exists", dir)
		}
	}
}

func initSpec(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `# The spec of the box %q, read by zk. Obsidian does not show this
# file; edit it in a text editor.
name = %q
hub = %q
base = %q
cards = "cards"
sources = "sources"

# With [fields] empty, every field zk writes has a zk_ prefix (zk_kind,
# zk_status, zk_created), so the box collides with nothing else in the
# vault. To write onto fields the vault already has, map them below, one
# line per field, zk's name = the vault's name. For example:
#
#   kind = "kind"
#   created = { name = "created_at", format = "date" }
#
`, name, name, name+" Hub.md", name+" Base.base")
	for _, line := range wrap("zk can map "+strings.Join(canonicalNames(), ", ")+
		`. created and published take a format, "date" or "datetime". A mapped field needs the same name in the base, and `+
		"`zk check --fix-types` to set its type in Obsidian.", 70) {
		b.WriteString("# " + line + "\n")
	}
	b.WriteString("[fields]\n")
	return b.String()
}

func wrap(text string, width int) []string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(text) {
		if line != "" && len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += w
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func initHub(name string) string {
	base := name + " Base.base"
	return fmt.Sprintf(`# %[1]s Hub

The entrance to this slip box. Everything an agent needs is here in plain text; the views below are embedded from the base.

## What is here

| What | Where |
|---|---|
| Cards | `+"`cards/zk-<key>.md`"+` |
| Sources | `+"`sources/<slug>.md`"+` |
| The box's spec | `+"`zk.toml`"+` (Obsidian does not show it; open it in a text editor) |
| The base | `+"`%[2]s`"+` |

A card holds one thought, at a permanent key (`+"`2a1`"+`). Its kind says what sort of statement it is: claim, fact, idea, question, quote, report or hub. Its status says how far along it is: `+"`raw`"+` (captured), `+"`proposed`"+` (placed and linked), `+"`linked`"+` (the owner agrees with its place).

## Working the box

- Make a card: `+"`zk new idea \"A thought\"`"+`, or `+"`zk new claim \"What they said\" --under 2`"+`.
- List it: `+"`zk list --tree`"+`; one family: `+"`zk list --under 2`"+`.
- Check it: `+"`zk check`"+`, and `+"`zk fix`"+` for the derived fields.

## Raw

![[%[2]s#Raw]]

## Proposed

![[%[2]s#Proposed]]

## All

![[%[2]s#All]]
`, name, base)
}

func initBase(m Mapping) string {
	class, kind, status, sort := m.Field("class").Name, m.Field("kind").Name, m.Field("status").Name, m.Field("sort").Name
	view := func(title, filter string, cols ...string) string {
		var b strings.Builder
		fmt.Fprintf(&b, "  - type: table\n    name: %s\n", title)
		if filter != "" {
			fmt.Fprintf(&b, "    filters:\n      and:\n        - %s\n", filter)
		}
		b.WriteString("    order:\n")
		for _, c := range cols {
			fmt.Fprintf(&b, "      - %s\n", c)
		}
		fmt.Fprintf(&b, "    sort:\n      - property: %s\n        direction: ASC\n", sort)
		return b.String()
	}
	return fmt.Sprintf("filters:\n  and:\n    - %s == %q\nformulas:\n  nickname: aliases[0]\nviews:\n", class, m.ClassValue(ClassCard)) +
		view("All", "", "file.name", "formula.nickname", kind, status) +
		view("Raw", fmt.Sprintf("%s == %q", status, "raw"), "file.name", "formula.nickname", kind) +
		view("Proposed", fmt.Sprintf("%s == %q", status, "proposed"), "file.name", "formula.nickname", kind)
}
