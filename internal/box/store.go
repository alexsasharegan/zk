package box

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/alexsasharegan/zk/internal/frontmatter"
	"github.com/alexsasharegan/zk/internal/key"
)

// Note is one Markdown file in the box.
type Note struct {
	Path   string // absolute
	Stem   string // file name without .md
	Key    string // for a card: the key in the file name ("" if not in the namespace)
	Raw    []byte // the whole file
	Doc    *frontmatter.Doc
	Fields map[string]any
	Err    error // the front matter could not be read
}

// Body is the note's text after the front matter.
func (n *Note) Body() string {
	if n.Doc == nil {
		return string(n.Raw)
	}
	return string(n.Doc.Body())
}

// Store is a box read from disk: its cards and sources.
type Store struct {
	Spec    *Spec
	Cards   []*Note          // every .md file in the cards folder
	ByKey   map[string]*Note // cards with a valid key
	Keys    []string         // valid keys, depth first
	Sources map[string]*Note // by stem
	names   map[string]bool  // note and file names in the vault, for links
}

// ArgError is an error in what the caller asked for (no such card, a bad
// key); the command line reports it as a usage error.
type ArgError struct{ Msg string }

func (e ArgError) Error() string { return e.Msg }

func argErr(format string, a ...any) error { return ArgError{fmt.Sprintf(format, a...)} }

// Open reads the box's cards and sources.
func Open(spec *Spec) (*Store, error) {
	s := &Store{Spec: spec, ByKey: map[string]*Note{}, Sources: map[string]*Note{}}
	cards, err := readNotes(spec.Cards)
	if err != nil {
		return nil, err
	}
	for _, n := range cards {
		n.Key = key.FromStem(spec.KeyPrefix, n.Stem)
		s.Cards = append(s.Cards, n)
		if key.Valid(n.Key) {
			s.ByKey[n.Key] = n
			s.Keys = append(s.Keys, n.Key)
		}
	}
	sort.Slice(s.Keys, func(i, j int) bool { return key.Less(s.Keys[i], s.Keys[j]) })
	sources, err := readNotes(spec.Sources)
	if err != nil {
		return nil, err
	}
	for _, n := range sources {
		s.Sources[n.Stem] = n
	}
	return s, nil
}

func readNotes(dir string) ([]*Note, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		n, err := readNote(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func readNote(p string) (*Note, error) {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	n := &Note{Path: p, Stem: strings.TrimSuffix(filepath.Base(p), ".md"), Raw: raw}
	if n.Doc, n.Err = frontmatter.Parse(raw); n.Err == nil {
		n.Fields, n.Err = n.Doc.Fields()
	}
	return n, nil
}

// Rel is a path relative to the vault, for messages.
func (s *Store) Rel(p string) string {
	if r, err := filepath.Rel(s.Spec.Vault, p); err == nil {
		return filepath.ToSlash(r)
	}
	return p
}

// CardPath is where the card with key k lives.
func (s *Store) CardPath(k string) string {
	return filepath.Join(s.Spec.Cards, key.Stem(s.Spec.KeyPrefix, k)+".md")
}

// SourcePath is where the source with slug lives.
func (s *Store) SourcePath(slug string) string {
	return filepath.Join(s.Spec.Sources, slug+".md")
}

var (
	wikilink = regexp.MustCompile(`\[\[([^\]|#]+)(?:[#|][^\]]*)?\]\]`)
	h1       = regexp.MustCompile(`(?m)^# (.+?)\s*$`)
)

// Links are the wikilink targets in text, in order, with repeats.
func Links(text string) []string {
	var out []string
	for _, m := range wikilink.FindAllStringSubmatch(text, -1) {
		out = append(out, strings.TrimSpace(m[1]))
	}
	return out
}

// H1 is the note's title, its first "# " heading, or "".
func H1(body string) string {
	if m := h1.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

// Resolves reports whether a wikilink target names a note or file in the
// vault. Obsidian resolves by name, so a path's last element is enough.
func (s *Store) Resolves(target string) bool {
	if s.names == nil {
		s.names = map[string]bool{}
		_ = filepath.WalkDir(s.Spec.Vault, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") && p != s.Spec.Vault {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() {
				s.names[d.Name()] = true
				s.names[strings.TrimSuffix(d.Name(), ".md")] = true
			}
			return nil
		})
	}
	t := path.Base(filepath.ToSlash(target))
	return s.names[t] || s.names[strings.TrimSuffix(t, ".md")]
}

// LinkKind says what a link target is in this box: card, source or other.
func (s *Store) LinkKind(target string) string {
	if _, ok := s.ByKey[key.FromStem(s.Spec.KeyPrefix, target)]; ok {
		return "card"
	}
	if _, ok := s.Sources[target]; ok {
		return "source"
	}
	return "other"
}

// Nickname is a card's aliases[0], else its H1.
func Nickname(n *Note) string {
	if al := stringList(n.Fields["aliases"]); len(al) > 0 {
		return al[0]
	}
	return H1(n.Body())
}

// stringList reads a front matter list of text.
func stringList(v any) []string {
	switch v := v.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			} else {
				out = append(out, fmt.Sprint(e))
			}
		}
		return out
	}
	return nil
}
