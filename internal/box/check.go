package box

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/alexsasharegan/zk/internal/frontmatter"
	"github.com/alexsasharegan/zk/internal/key"
)

// Statuses and forms.
var (
	CardStatuses   = []string{"raw", "proposed", "linked"}
	SourceStatuses = []string{"raw", "read", "extracted"}
	Forms          = []string{"article", "video", "paper", "book", "thread", "other"}
)

// Finding is one problem the check found.
type Finding struct {
	Path    string `json:"path"` // relative to the vault
	Key     string `json:"key,omitempty"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Warning bool   `json:"warning"`
	Fixable bool   `json:"fixable"` // zk check --fix repairs it
}

// Check checks the box's cards and sources. With strict, a duplicate
// nickname or a skipped key is an error, not a warning.
func (s *Store) Check(strict bool) []Finding {
	var out []Finding
	m := s.Spec.Mapping
	add := func(n *Note, k, rule, format string, a ...any) *Finding {
		out = append(out, Finding{Path: s.Rel(n.Path), Key: k, Rule: rule, Message: fmt.Sprintf(format, a...)})
		return &out[len(out)-1]
	}

	inbound := map[string]map[string]bool{} // card key -> stems linking to it
	link := func(to, from string) {
		if inbound[to] == nil {
			inbound[to] = map[string]bool{}
		}
		inbound[to][from] = true
	}
	notes := append([]*Note(nil), s.Cards...)
	for _, n := range sortedSources(s) {
		notes = append(notes, n)
	}
	for _, n := range notes {
		if n.Err != nil {
			add(n, n.Key, "front-matter", "%v", n.Err)
			continue
		}
		for _, t := range Links(string(n.Raw)) {
			if k := key.FromStem(s.Spec.KeyPrefix, t); k != "" {
				link(k, n.Stem)
			}
			if !s.Resolves(t) {
				add(n, n.Key, "link", "[[%s]] does not resolve", t)
			}
		}
	}
	for _, src := range s.Sources {
		if src.Err == nil {
			for _, k := range stringList(src.Fields[m.Field("extracted").Name]) {
				link(k, src.Stem)
			}
		}
	}

	for _, n := range s.Cards {
		if n.Err == nil {
			s.checkCard(n, add)
		}
	}
	for _, n := range sortedSources(s) {
		if n.Err == nil {
			s.checkSource(n, add)
		}
	}

	nicknames := map[string][]string{}
	for _, k := range s.Keys {
		n := s.ByKey[k]
		if n.Err != nil {
			continue
		}
		from := inbound[k]
		delete(from, n.Stem)
		if st, _ := n.Fields[m.Field("status").Name].(string); st != "raw" && len(from) == 0 {
			add(n, k, "orphan", "no card or source links to this card")
		}
		if nick := Nickname(n); nick != "" {
			fold := strings.ToLower(nick)
			nicknames[fold] = append(nicknames[fold], k)
		}
		if prev := key.Previous(k); prev != "" {
			if _, ok := s.ByKey[prev]; !ok {
				add(n, k, "gap", "card %s does not exist (a skipped key; harmless if you deleted it)", prev).Warning = !strict
			}
		}
	}
	for nick, ks := range nicknames {
		if len(ks) > 1 {
			add(s.ByKey[ks[0]], ks[0], "nickname", "%q is the nickname of cards %s", nick, strings.Join(ks, ", ")).Warning = !strict
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Warning != out[j].Warning {
			return !out[i].Warning
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func sortedSources(s *Store) []*Note {
	stems := make([]string, 0, len(s.Sources))
	for st := range s.Sources {
		stems = append(stems, st)
	}
	sort.Strings(stems)
	out := make([]*Note, len(stems))
	for i, st := range stems {
		out[i] = s.Sources[st]
	}
	return out
}

type adder func(n *Note, k, rule, format string, a ...any) *Finding

func (s *Store) checkCard(n *Note, add adder) {
	m := s.Spec.Mapping
	prefix := s.Spec.KeyPrefix
	if n.Key == "" {
		add(n, "", "key", "the file name must start with %s-", prefix)
		return
	}
	k := n.Key
	if !key.Valid(k) {
		add(n, k, "key", "%q is not a valid key (alternating number and letter segments, no leading zero)", k)
		return
	}
	fixable := func(f *Finding) { f.Fixable = true }

	classField := m.Field("class").Name
	switch v := n.Fields[classField]; {
	case v == nil:
		fixable(add(n, k, "front-matter", "missing %s", classField))
	case v != m.ClassValue(ClassCard):
		add(n, k, "front-matter", "%s is %v, want %s", classField, v, m.ClassValue(ClassCard))
		return
	}
	fields := copyFields(n.Fields)
	fields[classField] = m.ClassValue(ClassCard)
	c, err := m.Card(fields)
	if err != nil {
		add(n, k, "front-matter", "%v", err)
	}

	derived := map[string]bool{"key": true, "level": true, "sort": true}
	for _, f := range []string{"key", "level", "sort", "kind", "status", "created"} {
		if n.Fields[m.Field(f).Name] == nil {
			add(n, k, "front-matter", "missing %s", m.Field(f).Name).Fixable = derived[f]
		}
	}
	title := H1(n.Body())
	aliases := stringList(n.Fields["aliases"])
	tags := stringList(n.Fields["tags"])
	switch {
	case title == "":
		add(n, k, "title", "no # H1: the card's title is its first \"# \" heading")
		if len(aliases) == 0 {
			add(n, k, "front-matter", "missing aliases")
		}
	case len(aliases) == 0:
		fixable(add(n, k, "front-matter", "missing aliases"))
	case aliases[0] != title:
		fixable(add(n, k, "derived", "aliases[0] is %q, the H1 says %q", aliases[0], title))
	}
	if !contains(tags, key.Tag(prefix, k)) {
		fixable(add(n, k, "derived", "tags should include %s", key.Tag(prefix, k)))
	}
	keyField := m.Field("key").Name
	if v, present := n.Fields[keyField]; present {
		if _, isText := v.(string); !isText {
			fixable(add(n, k, "key", "quote the key (%s: \"%s\") so Obsidian reads it as text, not a number", keyField, k))
		} else if c.Key != k {
			fixable(add(n, k, "key", "%s is %q, the file name says %q", keyField, c.Key, k))
		}
	}
	if n.Fields[m.Field("level").Name] != nil && c.Level != key.Level(k) {
		fixable(add(n, k, "derived", "%s should be %d", m.Field("level").Name, key.Level(k)))
	}
	if n.Fields[m.Field("sort").Name] != nil && c.Sort != key.Sort(k) {
		fixable(add(n, k, "derived", "%s should be %s", m.Field("sort").Name, key.Sort(k)))
	}
	kind, known := s.Spec.Kinds[c.Kind]
	if c.Kind != "" && !known {
		add(n, k, "front-matter", "%s must be one of %s", m.Field("kind").Name, strings.Join(s.Spec.KindNames(), ", "))
	}
	if c.Status != "" && !contains(CardStatuses, c.Status) {
		add(n, k, "front-matter", "%s must be one of %s", m.Field("status").Name, strings.Join(CardStatuses, ", "))
	}
	if p := key.Parent(k); p != "" {
		if _, ok := s.ByKey[p]; !ok {
			add(n, k, "parent", "parent card %s does not exist", p)
		}
	}
	if kind.NeedsSource {
		linked := false
		for _, t := range Links(string(n.Raw)) {
			if _, ok := s.Sources[t]; ok {
				linked = true
			}
		}
		if !linked {
			add(n, k, "source", "a %s card must link to its source page in %s (a wikilink in the body)", c.Kind, s.Rel(s.Spec.Sources))
		}
	}
	if kind.NeedsLink {
		_, connections, _ := strings.Cut(n.Body(), "## Connections")
		if len(Links(connections)) == 0 {
			add(n, k, "hub", "a %s card needs a ## Connections section with at least one link", c.Kind)
		}
	}
}

func (s *Store) checkSource(n *Note, add adder) {
	m := s.Spec.Mapping
	classField := m.Field("class").Name
	if v := n.Fields[classField]; v != m.ClassValue(ClassSource) {
		add(n, "", "front-matter", "%s is %v, want %s", classField, v, m.ClassValue(ClassSource))
		return
	}
	src, err := m.Source(n.Fields)
	if err != nil {
		add(n, "", "front-matter", "%v", err)
	}
	for _, f := range []string{"title", "provenance", "form", "status", "created"} {
		if n.Fields[m.Field(f).Name] == nil {
			add(n, "", "front-matter", "missing %s", m.Field(f).Name)
		}
	}
	if src.Form != "" && !contains(Forms, src.Form) {
		add(n, "", "front-matter", "%s must be one of %s", m.Field("form").Name, strings.Join(Forms, ", "))
	}
	if src.Status != "" && !contains(SourceStatuses, src.Status) {
		add(n, "", "front-matter", "%s must be one of %s", m.Field("status").Name, strings.Join(SourceStatuses, ", "))
	}
	if src.Status == "extracted" && len(src.Extracted) == 0 {
		add(n, "", "front-matter", "status extracted needs %s (the keys of the cards)", m.Field("extracted").Name)
	}
	for _, k := range src.Extracted {
		if _, ok := s.ByKey[k]; !ok {
			add(n, "", "extracted", "card %s does not exist", k)
		}
	}
}

// Fix writes the derived fields of the given cards (all when keys is
// empty): the class when missing, the key, level and sort from the file
// name, the key's tag, and aliases[0] from the H1. It returns the cards it
// changed, depth first.
func (s *Store) Fix(keys []string) ([]string, error) {
	if len(keys) == 0 {
		keys = s.Keys
	}
	for _, k := range keys {
		if _, ok := s.ByKey[k]; !ok {
			return nil, argErr("no card %q", k)
		}
	}
	m := s.Spec.Mapping
	prefix := s.Spec.KeyPrefix
	var changed []string
	for _, k := range keys {
		n := s.ByKey[k]
		if n.Err != nil {
			continue
		}
		title := H1(n.Body())
		err := n.Doc.Update(func(fm map[string]any) {
			if fm[m.Field("class").Name] == nil {
				fm[m.Field("class").Name] = m.ClassValue(ClassCard)
			}
			fm[m.Field("key").Name] = frontmatter.Quoted(k)
			fm[m.Field("level").Name] = key.Level(k)
			fm[m.Field("sort").Name] = frontmatter.Quoted(key.Sort(k))
			tags := []string{key.Tag(prefix, k)}
			for _, t := range stringList(fm["tags"]) {
				if t != "" && !strings.HasPrefix(t, prefix+"/") {
					tags = append(tags, t)
				}
			}
			fm["tags"] = tags
			if title != "" {
				aliases := []string{title}
				old := stringList(fm["aliases"])
				for i, a := range old {
					if i > 0 && a != "" && a != title {
						aliases = append(aliases, a)
					}
				}
				fm["aliases"] = aliases
			}
		})
		if err != nil {
			return changed, fmt.Errorf("%s: %w", s.Rel(n.Path), err)
		}
		out, err := n.Doc.Bytes()
		if err != nil {
			return changed, err
		}
		if bytes.Equal(out, n.Raw) {
			continue
		}
		if err := frontmatter.WriteFile(n.Path, n.Doc); err != nil {
			return changed, err
		}
		n.Raw = out
		n.Fields, _ = n.Doc.Fields()
		changed = append(changed, k)
	}
	return changed, nil
}

func copyFields(fm map[string]any) map[string]any {
	out := make(map[string]any, len(fm))
	for k, v := range fm {
		out[k] = v
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
