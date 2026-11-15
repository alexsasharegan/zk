package box

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alexsasharegan/zk/internal/frontmatter"
	"github.com/alexsasharegan/zk/internal/key"
)

// ErrExists is returned when a note to be made is already there.
var ErrExists = errors.New("already exists")

// CleanTitle joins title words and collapses spaces.
func CleanTitle(words []string) (string, error) {
	t := strings.Join(strings.Fields(strings.Join(words, " ")), " ")
	if t == "" {
		return "", argErr("a title cannot be empty")
	}
	return t, nil
}

// NewCard is what zk new makes.
type NewCard struct {
	Kind  string
	Title string
	Under string // the parent's key, or "" for a new root
	Agent bool   // an agent made it: sets authored_by
	Now   time.Time
}

// PlanCard returns the key and path a new card would get, without writing.
func (s *Store) PlanCard(c NewCard) (string, string, error) {
	if _, ok := s.Spec.Kinds[c.Kind]; !ok {
		return "", "", argErr("%q is not a kind in this box (%s)", c.Kind, strings.Join(s.Spec.KindNames(), ", "))
	}
	var k string
	if c.Under != "" {
		if _, ok := s.ByKey[c.Under]; !ok {
			return "", "", argErr("no card %q to go under", c.Under)
		}
		k = key.NextChild(c.Under, s.Keys)
	} else {
		k = key.NextRoot(s.Keys)
	}
	return k, s.CardPath(k), nil
}

// MakeCard writes a new card: front matter in a fixed order, then the title
// as an H1 and an empty Connections section.
func (s *Store) MakeCard(c NewCard) (string, string, error) {
	k, p, err := s.PlanCard(c)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(p); err == nil {
		return "", "", fmt.Errorf("%s: %w", s.Rel(p), ErrExists)
	}
	m := s.Spec.Mapping
	doc, _ := frontmatter.Parse([]byte("# " + c.Title + "\n\n\n## Connections\n"))
	authored := ""
	if c.Agent {
		authored = "agent"
	}
	created := Date{Time: c.Now, Format: m.Field("created").Format}
	for _, f := range []struct {
		name string
		v    any
	}{
		{m.Field("class").Name, m.ClassValue(ClassCard)},
		{m.Field("kind").Name, c.Kind},
		{m.Field("key").Name, frontmatter.Quoted(k)},
		{m.Field("level").Name, key.Level(k)},
		{m.Field("sort").Name, frontmatter.Quoted(key.Sort(k))},
		{"aliases", []string{c.Title}},
		{m.Field("status").Name, "raw"},
		{m.Field("created").Name, created},
		{m.Field("authored_by").Name, authored},
		{"tags", []string{key.Tag(s.Spec.KeyPrefix, k)}},
	} {
		if f.v == "" {
			continue
		}
		if err := doc.Append(f.name, f.v); err != nil {
			return "", "", err
		}
	}
	if err := os.MkdirAll(s.Spec.Cards, 0o755); err != nil {
		return "", "", err
	}
	if err := frontmatter.WriteFile(p, doc); err != nil {
		return "", "", err
	}
	return k, p, nil
}

// Retitle changes a card's H1 and aliases[0] together. It refuses a
// nickname another card has. With links, it rewrites old
// "[[zk-K|(K) Old title]]" links in cards and sources, and returns how many.
func (s *Store) Retitle(k, title string, links bool) (old string, rewrote, files int, err error) {
	n, ok := s.ByKey[k]
	if !ok {
		return "", 0, 0, argErr("no card %q", k)
	}
	if n.Err != nil {
		return "", 0, 0, fmt.Errorf("%s: %w", s.Rel(n.Path), n.Err)
	}
	for _, other := range s.Keys {
		if other != k && strings.EqualFold(Nickname(s.ByKey[other]), title) {
			return "", 0, 0, fmt.Errorf("card %s already has the nickname %q", other, title)
		}
	}
	old = Nickname(n)
	body := n.Body()
	if loc := regexp.MustCompile(`(?m)^# .+$`).FindStringIndex(body); loc != nil {
		body = body[:loc[0]] + "# " + title + body[loc[1]:]
	} else {
		body = "# " + title + "\n\n" + strings.TrimLeft(body, "\n")
	}
	n.Doc.SetBody([]byte(body))
	if err := frontmatter.WriteFile(n.Path, n.Doc); err != nil {
		return "", 0, 0, err
	}
	if n.Raw, err = n.Doc.Bytes(); err != nil {
		return "", 0, 0, err
	}
	if _, err := s.Fix([]string{k}); err != nil {
		return "", 0, 0, err
	}
	if !links || old == "" {
		return old, 0, 0, nil
	}
	stem := key.Stem(s.Spec.KeyPrefix, k)
	pat := regexp.MustCompile(`(?i)\[\[` + regexp.QuoteMeta(stem) + `\|\(` + regexp.QuoteMeta(k) + `\) ` + regexp.QuoteMeta(old) + `\]\]`)
	repl := "[[" + stem + "|(" + k + ") " + title + "]]"
	for _, note := range append(append([]*Note(nil), s.Cards...), sortedSources(s)...) {
		data, err := os.ReadFile(note.Path)
		if err != nil {
			return old, rewrote, files, err
		}
		matches := pat.FindAllIndex(data, -1)
		if len(matches) == 0 {
			continue
		}
		data = pat.ReplaceAllLiteral(data, []byte(repl))
		if err := writeRaw(note.Path, data); err != nil {
			return old, rewrote, files, err
		}
		rewrote += len(matches)
		files++
	}
	return old, rewrote, files, nil
}

// writeRaw replaces a file's bytes, keeping its permissions.
func writeRaw(p string, data []byte) error {
	info, err := os.Stat(p)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, info.Mode().Perm())
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// NewSource is what zk source new makes.
type NewSource struct {
	Slug       string
	Title      string // default: from the slug
	Provenance string // a URL, a wikilink to a vault note, or a description
	Author     string
	Via        string
	Form       string
	Now        time.Time
}

// MakeSource writes a source page.
func (s *Store) MakeSource(src NewSource) (string, error) {
	if !slugPattern.MatchString(src.Slug) {
		return "", argErr("%q is not a slug (lower-case words joined by hyphens)", src.Slug)
	}
	if strings.TrimSpace(src.Provenance) == "" {
		return "", argErr("a source needs --provenance: a URL, a [[link]] to a note, or a description")
	}
	if src.Form == "" {
		src.Form = "article"
	}
	if !contains(Forms, src.Form) {
		return "", argErr("form %q must be one of %s", src.Form, strings.Join(Forms, ", "))
	}
	p := s.SourcePath(src.Slug)
	if _, err := os.Stat(p); err == nil {
		return "", fmt.Errorf("%s: %w", s.Rel(p), ErrExists)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	title := src.Title
	if title == "" {
		t := strings.ReplaceAll(src.Slug, "-", " ")
		title = strings.ToUpper(t[:1]) + t[1:]
	}
	m := s.Spec.Mapping
	doc, _ := frontmatter.Parse([]byte("# " + title + "\n\n"))
	for _, f := range []struct {
		name string
		v    any
	}{
		{m.Field("class").Name, m.ClassValue(ClassSource)},
		{m.Field("title").Name, title},
		{m.Field("provenance").Name, src.Provenance},
		{m.Field("author").Name, src.Author},
		{m.Field("via").Name, src.Via},
		{m.Field("form").Name, src.Form},
		{m.Field("status").Name, "raw"},
		{m.Field("created").Name, Date{Time: src.Now, Format: m.Field("created").Format}},
	} {
		if f.v == "" {
			continue
		}
		if err := doc.Append(f.name, f.v); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(s.Spec.Sources, 0o755); err != nil {
		return "", err
	}
	return p, frontmatter.WriteFile(p, doc)
}

// SourceDone marks a source extracted and adds the cards it made to any it
// already lists. It returns the full list, depth first.
func (s *Store) SourceDone(slug string, keys []string) ([]string, error) {
	n, ok := s.Sources[slug]
	if !ok {
		return nil, argErr("no source %q", slug)
	}
	if n.Err != nil {
		return nil, fmt.Errorf("%s: %w", s.Rel(n.Path), n.Err)
	}
	for _, k := range keys {
		if _, ok := s.ByKey[k]; !ok {
			return nil, argErr("no card %q", k)
		}
	}
	m := s.Spec.Mapping
	var merged []string
	var readErr error
	err := n.Doc.Update(func(fm map[string]any) {
		have := map[string]bool{}
		for _, k := range append(stringList(fm[m.Field("extracted").Name]), keys...) {
			if !have[k] {
				have[k] = true
				merged = append(merged, k)
			}
		}
		sort.Slice(merged, func(i, j int) bool { return key.Less(merged[i], merged[j]) })
		if fm[m.Field("class").Name] != m.ClassValue(ClassSource) {
			readErr = fmt.Errorf("%s is not a source (%s is %v)", s.Rel(n.Path), m.Field("class").Name, fm[m.Field("class").Name])
			return
		}
		fm[m.Field("status").Name] = "extracted"
		fm[m.Field("extracted").Name] = merged
	})
	if err == nil {
		err = readErr
	}
	if err != nil {
		return nil, err
	}
	return merged, frontmatter.WriteFile(n.Path, n.Doc)
}

// KeyInfo is what zk show says about a key.
type KeyInfo struct {
	Key          string `json:"key"`
	Level        int    `json:"level"`
	Sort         string `json:"sort"`
	Tag          string `json:"tag"`
	File         string `json:"file"`
	Exists       bool   `json:"exists"`
	Title        string `json:"title,omitempty"`
	Parent       string `json:"parent,omitempty"`
	ParentExists *bool  `json:"parent_exists,omitempty"`
	NextChild    string `json:"next_child,omitempty"`
	Links        []Link `json:"links"`
}

// Link is one link to or from a card.
type Link struct {
	Direction string `json:"direction"` // "out" or "in"
	Target    string `json:"target"`    // the other note's stem
	Kind      string `json:"kind"`      // card, source or other
	Via       string `json:"via"`       // link or extracted
}

// Show explains a key, and for a card that exists, lists its links.
func (s *Store) Show(k string) (KeyInfo, error) {
	if !key.Valid(k) {
		return KeyInfo{}, argErr("%q is not a valid key", k)
	}
	info := KeyInfo{Key: k, Level: key.Level(k), Sort: key.Sort(k), Tag: key.Tag(s.Spec.KeyPrefix, k), File: s.Rel(s.CardPath(k)), Links: []Link{}}
	n, exists := s.ByKey[k]
	info.Exists = exists
	if p := key.Parent(k); p != "" {
		_, ok := s.ByKey[p]
		info.Parent, info.ParentExists = p, &ok
	}
	if !exists {
		return info, nil
	}
	info.Title = Nickname(n)
	info.NextChild = key.NextChild(k, s.Keys)
	stem := key.Stem(s.Spec.KeyPrefix, k)
	seen := map[string]bool{}
	for _, t := range Links(string(n.Raw)) {
		if t != stem && !seen[t] {
			seen[t] = true
			info.Links = append(info.Links, Link{Direction: "out", Target: t, Kind: s.LinkKind(t), Via: "link"})
		}
	}
	m := s.Spec.Mapping
	in := func(other *Note, kind string) {
		if other == n || other.Err != nil {
			return
		}
		if contains(Links(string(other.Raw)), stem) {
			info.Links = append(info.Links, Link{Direction: "in", Target: other.Stem, Kind: kind, Via: "link"})
		}
		if kind == "source" && contains(stringList(other.Fields[m.Field("extracted").Name]), k) {
			info.Links = append(info.Links, Link{Direction: "in", Target: other.Stem, Kind: kind, Via: "extracted"})
		}
	}
	for _, other := range s.Cards {
		in(other, "card")
	}
	for _, other := range sortedSources(s) {
		in(other, "source")
	}
	return info, nil
}

// Row is one card in zk list.
type Row struct {
	Key    string `json:"key"`
	Level  int    `json:"level"`
	Status string `json:"status"`
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Path   string `json:"path"`
}

// Filter narrows zk list.
type Filter struct {
	Status, Kind, Under string
}

// List returns the cards depth first, filtered.
func (s *Store) List(f Filter) ([]Row, error) {
	if f.Under != "" {
		if _, ok := s.ByKey[f.Under]; !ok {
			return nil, argErr("no card %q", f.Under)
		}
	}
	m := s.Spec.Mapping
	var rows []Row
	for _, k := range s.Keys {
		n := s.ByKey[k]
		status, _ := n.Fields[m.Field("status").Name].(string)
		kind, _ := n.Fields[m.Field("kind").Name].(string)
		if (f.Status != "" && status != f.Status) || (f.Kind != "" && kind != f.Kind) || (f.Under != "" && !key.IsAncestor(f.Under, k)) {
			continue
		}
		rows = append(rows, Row{Key: k, Level: key.Level(k), Status: status, Kind: kind, Title: Nickname(n), Path: s.Rel(n.Path)})
	}
	return rows, nil
}
