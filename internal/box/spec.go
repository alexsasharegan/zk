// Package box reads a box's spec (zk.toml) and maps the box's front matter
// onto a vault's field names (ADR 261115-0tp4).
package box

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/alexsasharegan/zk/internal/config"
)

// Kind is one sort of card: its description, shown in the terminal UI, and
// the rule the check applies.
type Kind struct {
	Description string `toml:"description"`
	// NeedsSource: the card must link to a source page.
	NeedsSource bool `toml:"needs_source"`
	// NeedsLink: the card must link to at least one other card.
	NeedsLink bool `toml:"needs_link"`
}

// DefaultKinds are used when a spec defines no [kinds].
var DefaultKinds = map[string]Kind{
	"claim":    {Description: "Someone asserts this. Say who, and where. It says nothing about whether it holds.", NeedsSource: true},
	"fact":     {Description: "You hold this with strong confidence. Name its scope and the evidence.", NeedsSource: true},
	"idea":     {Description: "Your own thought: something that surprised you, or a connection you noticed."},
	"question": {Description: "Something you are curious about and cannot answer yet."},
	"quote":    {Description: "Someone's exact words, with who said them and where.", NeedsSource: true},
	"report":   {Description: "Something you did, saw or lived through, in your own words.", NeedsSource: true},
	"hub":      {Description: "An entrance for a person or a topic: a short description, then links.", NeedsLink: true},
}

// Spec is a loaded zk.toml. Paths are absolute.
type Spec struct {
	// Path is the spec file.
	Path string
	// Dir is the box's managed folder, the folder that holds the spec.
	Dir string
	// Vault is the Obsidian vault the box lives in: the nearest folder at or
	// above Dir that holds .obsidian/.
	Vault string

	Name      string
	Hub       string // the hub note; "" when the spec names none
	Base      string // the base; "" when the spec names none
	Cards     string
	Sources   string
	KeyPrefix string
	Kinds     map[string]Kind
	Mapping   Mapping
}

// KindNames returns the kinds in sorted order.
func (s *Spec) KindNames() []string {
	names := make([]string, 0, len(s.Kinds))
	for n := range s.Kinds {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// rawSpec is the TOML shape of zk.toml.
type rawSpec struct {
	Name      string                    `toml:"name"`
	Hub       string                    `toml:"hub"`
	Base      string                    `toml:"base"`
	Cards     string                    `toml:"cards"`
	Sources   string                    `toml:"sources"`
	KeyPrefix string                    `toml:"key_prefix"`
	Kinds     map[string]Kind           `toml:"kinds"`
	Classes   map[string]string         `toml:"classes"`
	Fields    map[string]toml.Primitive `toml:"fields"`
}

var (
	kindName  = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	keyPrefix = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
)

// Load reads and validates the spec at path.
func Load(path string) (*Spec, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	var raw rawSpec
	md, err := toml.DecodeFile(path, &raw)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("no spec at %s (%s)", path, config.SpecHint)
		}
		return nil, fmt.Errorf("reading the box spec %s: %w", path, err)
	}
	fail := func(format string, a ...any) (*Spec, error) {
		return nil, fmt.Errorf("the box spec %s: %s", path, fmt.Sprintf(format, a...))
	}

	mapping, err := parseMapping(md, raw.Fields, raw.Classes)
	if err != nil {
		return fail("%v", err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		// parseMapping has decoded [fields], so anything left is unknown.
		keys := make([]string, len(undecoded))
		for i, k := range undecoded {
			keys[i] = k.String()
		}
		return fail("unknown keys: %s", strings.Join(keys, ", "))
	}

	s := &Spec{Path: path, Dir: filepath.Dir(path), Name: raw.Name, KeyPrefix: raw.KeyPrefix, Mapping: mapping}
	if s.Vault, err = findVault(s.Dir); err != nil {
		return fail("%v", err)
	}
	if s.Name == "" {
		return fail("name is required")
	}
	if s.KeyPrefix == "" {
		s.KeyPrefix = "zk"
	}
	if !keyPrefix.MatchString(s.KeyPrefix) {
		return fail("key_prefix %q must be lower-case letters and digits, starting with a letter", s.KeyPrefix)
	}

	paths := []struct {
		dst      *string
		value    string
		key, def string
	}{
		{&s.Cards, raw.Cards, "cards", "cards"},
		{&s.Sources, raw.Sources, "sources", "sources"},
		{&s.Hub, raw.Hub, "hub", ""},
		{&s.Base, raw.Base, "base", ""},
	}
	for _, p := range paths {
		v := p.value
		if v == "" {
			v = p.def
		}
		if v == "" {
			continue
		}
		abs, err := s.inVault(v)
		if err != nil {
			return fail("%s: %v", p.key, err)
		}
		*p.dst = abs
	}
	if s.Cards == s.Sources {
		return fail("cards and sources must be different folders")
	}

	s.Kinds = raw.Kinds
	if len(s.Kinds) == 0 {
		s.Kinds = DefaultKinds
	}
	for name, k := range s.Kinds {
		if !kindName.MatchString(name) {
			return fail("kind %q must be lower-case letters, digits and hyphens, starting with a letter", name)
		}
		if strings.TrimSpace(k.Description) == "" {
			return fail("kind %q needs a description", name)
		}
	}
	return s, nil
}

// inVault resolves a spec path relative to the box folder and checks that it
// stays inside the vault.
func (s *Spec) inVault(p string) (string, error) {
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("%q must be relative to the box folder", p)
	}
	abs := filepath.Join(s.Dir, p)
	rel, err := filepath.Rel(s.Vault, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%q is outside the vault %s", p, s.Vault)
	}
	return abs, nil
}

// findVault returns the nearest folder at or above dir that holds .obsidian/.
func findVault(dir string) (string, error) {
	start := dir
	for {
		info, err := os.Stat(filepath.Join(dir, ".obsidian"))
		if err == nil && info.IsDir() {
			return dir, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside an Obsidian vault: no .obsidian folder at or above %s", start)
		}
		dir = parent
	}
}
