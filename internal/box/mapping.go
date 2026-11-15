package box

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Class is the class of a box note.
type Class string

// The classes of box notes. Trails are reserved and not mapped yet.
const (
	ClassCard   Class = "card"
	ClassSource Class = "source"
)

// Format is how a date field is written.
type Format string

// Date formats. A datetime is local time with no zone.
const (
	FormatDate     Format = "date"     // 2006-01-02
	FormatDatetime Format = "datetime" // 2006-01-02T15:04:05
)

func (f Format) layout() string {
	if f == FormatDate {
		return "2006-01-02"
	}
	return "2006-01-02T15:04:05"
}

// Field is where a canonical field is written in a vault.
type Field struct {
	Name   string
	Format Format // only for date fields
}

// canonicalField describes one of the box's own fields.
type canonicalField struct {
	date          bool
	defaultFormat Format
}

// canonical lists the box's fields (ADR 261115-0tp4). A field the spec does
// not map is written as zk_<name>.
var canonical = map[string]canonicalField{
	"class":       {},
	"kind":        {},
	"key":         {},
	"level":       {},
	"sort":        {},
	"status":      {},
	"created":     {date: true, defaultFormat: FormatDatetime},
	"authored_by": {},
	"title":       {},
	"author":      {},
	"published":   {date: true, defaultFormat: FormatDate},
	"provenance":  {},
	"via":         {},
	"form":        {},
	"extracted":   {},
	"series":      {},
}

// obsidianFields belong to Obsidian and are never mapped onto.
var obsidianFields = map[string]bool{"aliases": true, "tags": true}

var (
	fieldName  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	classValue = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// Mapping maps the box's canonical fields and class values onto a vault's.
type Mapping struct {
	fields  map[string]Field
	classes map[Class]string
}

// DefaultMapping writes every field as zk_<name>, with class values "card"
// and "source".
func DefaultMapping() Mapping {
	m, _ := newMapping(nil, nil)
	return m
}

// Field returns where a canonical field is written. It panics on a name that
// is not a canonical field, which is a programming error.
func (m Mapping) Field(name string) Field {
	f, ok := m.fields[name]
	if !ok {
		panic("box: not a canonical field: " + name)
	}
	return f
}

// ClassValue returns the value of the class field for a class.
func (m Mapping) ClassValue(c Class) string { return m.classes[c] }

// ClassOf returns the class of a note from its front matter.
func (m Mapping) ClassOf(fm map[string]any) (Class, bool) {
	v, ok := fm[m.Field("class").Name].(string)
	if !ok {
		return "", false
	}
	for c, cv := range m.classes {
		if v == cv {
			return c, true
		}
	}
	return "", false
}

func newMapping(fields map[string]Field, classes map[Class]string) (Mapping, error) {
	m := Mapping{fields: map[string]Field{}, classes: map[Class]string{ClassCard: "card", ClassSource: "source"}}
	for name, cf := range canonical {
		f := Field{Name: "zk_" + name}
		if cf.date {
			f.Format = cf.defaultFormat
		}
		m.fields[name] = f
	}
	for name, f := range fields {
		cf, ok := canonical[name]
		if !ok {
			return Mapping{}, fmt.Errorf("[fields]: %q is not a field zk writes (it writes %s)", name, strings.Join(canonicalNames(), ", "))
		}
		if !fieldName.MatchString(f.Name) {
			return Mapping{}, fmt.Errorf("[fields] %s: %q is not a usable field name", name, f.Name)
		}
		if obsidianFields[f.Name] {
			return Mapping{}, fmt.Errorf("[fields] %s: %q belongs to Obsidian and cannot be mapped onto", name, f.Name)
		}
		switch {
		case f.Format != "" && !cf.date:
			return Mapping{}, fmt.Errorf("[fields] %s: only date fields take a format", name)
		case f.Format == "" && cf.date:
			f.Format = cf.defaultFormat
		case f.Format != "" && f.Format != FormatDate && f.Format != FormatDatetime:
			return Mapping{}, fmt.Errorf("[fields] %s: format %q must be %q or %q", name, f.Format, FormatDate, FormatDatetime)
		}
		m.fields[name] = f
	}
	seen := map[string]string{}
	for _, name := range canonicalNames() {
		vault := m.fields[name].Name
		if other, dup := seen[vault]; dup {
			return Mapping{}, fmt.Errorf("[fields]: %s and %s both map to %q", other, name, vault)
		}
		seen[vault] = name
	}
	for c, v := range classes {
		if c != ClassCard && c != ClassSource {
			return Mapping{}, fmt.Errorf("[classes]: %q is not a class (card, source)", c)
		}
		if !classValue.MatchString(v) {
			return Mapping{}, fmt.Errorf("[classes] %s: %q must be lower-case letters, digits and hyphens", c, v)
		}
		m.classes[c] = v
	}
	if m.classes[ClassCard] == m.classes[ClassSource] {
		return Mapping{}, errors.New("[classes]: card and source need different values")
	}
	return m, nil
}

// parseMapping reads [fields] and [classes] from a spec. A [fields] entry is
// either a vault field name or a table {name, format}.
func parseMapping(md toml.MetaData, raw map[string]toml.Primitive, rawClasses map[string]string) (Mapping, error) {
	fields := map[string]Field{}
	for name, prim := range raw {
		var f Field
		switch md.Type("fields", name) {
		case "String":
			if err := md.PrimitiveDecode(prim, &f.Name); err != nil {
				return Mapping{}, err
			}
		case "Hash", "Inline Table":
			var t struct {
				Name   string `toml:"name"`
				Format string `toml:"format"`
			}
			if err := md.PrimitiveDecode(prim, &t); err != nil {
				return Mapping{}, err
			}
			f = Field{Name: t.Name, Format: Format(t.Format)}
		default:
			return Mapping{}, fmt.Errorf("[fields] %s: want a field name or a table {name, format}", name)
		}
		fields[name] = f
	}
	classes := map[Class]string{}
	for c, v := range rawClasses {
		classes[Class(c)] = v
	}
	return newMapping(fields, classes)
}

func canonicalNames() []string {
	names := make([]string, 0, len(canonical))
	for n := range canonical {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Date is a date field's value, with the format it is written in.
type Date struct {
	Time   time.Time
	Format Format
}

func (d Date) String() string { return d.Time.Format(d.Format.layout()) }

// Card is a card's front matter, in the box's own terms.
type Card struct {
	Kind       string
	Key        string
	Level      int
	Sort       string
	Status     string
	Created    time.Time
	AuthoredBy string
}

// Source is a source page's front matter, in the box's own terms.
type Source struct {
	Title      string
	Author     string
	Published  time.Time
	Provenance string
	Via        string
	Form       string
	Status     string
	Created    time.Time
	Extracted  []string
	Series     string
}

// ErrWrongClass is returned when a note is not of the class asked for.
var ErrWrongClass = errors.New("not a note of that class")

// PutCard writes a card into front matter through the mapping. Fields of the
// note that the box does not own are left alone; an empty value removes its
// field.
func (m Mapping) PutCard(fm map[string]any, c Card) {
	m.put(fm, "class", m.classes[ClassCard])
	m.put(fm, "kind", c.Kind)
	m.put(fm, "key", c.Key)
	m.putInt(fm, "level", c.Level)
	m.put(fm, "sort", c.Sort)
	m.put(fm, "status", c.Status)
	m.putDate(fm, "created", c.Created)
	m.put(fm, "authored_by", c.AuthoredBy)
}

// Card reads a card from front matter through the mapping.
func (m Mapping) Card(fm map[string]any) (Card, error) {
	if c, _ := m.ClassOf(fm); c != ClassCard {
		return Card{}, fmt.Errorf("%w: %s is not %q", ErrWrongClass, m.Field("class").Name, m.classes[ClassCard])
	}
	r := reader{m: m, fm: fm}
	c := Card{
		Kind:       r.str("kind"),
		Key:        r.str("key"),
		Level:      r.int("level"),
		Sort:       r.str("sort"),
		Status:     r.str("status"),
		Created:    r.date("created"),
		AuthoredBy: r.str("authored_by"),
	}
	return c, r.err()
}

// PutSource writes a source into front matter through the mapping.
func (m Mapping) PutSource(fm map[string]any, s Source) {
	m.put(fm, "class", m.classes[ClassSource])
	m.put(fm, "title", s.Title)
	m.put(fm, "author", s.Author)
	m.putDate(fm, "published", s.Published)
	m.put(fm, "provenance", s.Provenance)
	m.put(fm, "via", s.Via)
	m.put(fm, "form", s.Form)
	m.put(fm, "status", s.Status)
	m.putDate(fm, "created", s.Created)
	if len(s.Extracted) > 0 {
		fm[m.Field("extracted").Name] = append([]string(nil), s.Extracted...)
	} else {
		delete(fm, m.Field("extracted").Name)
	}
	m.put(fm, "series", s.Series)
}

// Source reads a source from front matter through the mapping.
func (m Mapping) Source(fm map[string]any) (Source, error) {
	if c, _ := m.ClassOf(fm); c != ClassSource {
		return Source{}, fmt.Errorf("%w: %s is not %q", ErrWrongClass, m.Field("class").Name, m.classes[ClassSource])
	}
	r := reader{m: m, fm: fm}
	s := Source{
		Title:      r.str("title"),
		Author:     r.str("author"),
		Published:  r.date("published"),
		Provenance: r.str("provenance"),
		Via:        r.str("via"),
		Form:       r.str("form"),
		Status:     r.str("status"),
		Created:    r.date("created"),
		Extracted:  r.list("extracted"),
		Series:     r.str("series"),
	}
	return s, r.err()
}

func (m Mapping) put(fm map[string]any, name, v string) {
	if v == "" {
		delete(fm, m.Field(name).Name)
		return
	}
	fm[m.Field(name).Name] = v
}

func (m Mapping) putInt(fm map[string]any, name string, v int) {
	if v == 0 {
		delete(fm, m.Field(name).Name)
		return
	}
	fm[m.Field(name).Name] = v
}

func (m Mapping) putDate(fm map[string]any, name string, t time.Time) {
	f := m.Field(name)
	if t.IsZero() {
		delete(fm, f.Name)
		return
	}
	fm[f.Name] = Date{Time: t, Format: f.Format}
}

// reader reads canonical fields from front matter, collecting type errors.
type reader struct {
	m    Mapping
	fm   map[string]any
	errs []string
}

func (r *reader) fail(name string, v any, want string) {
	r.errs = append(r.errs, fmt.Sprintf("%s: %v is not %s", r.m.Field(name).Name, v, want))
}

func (r *reader) err() error {
	if len(r.errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(r.errs, "; "))
}

func (r *reader) str(name string) string {
	switch v := r.fm[r.m.Field(name).Name].(type) {
	case nil:
		return ""
	case string:
		return v
	case int, int64, uint64:
		// An unquoted key such as 7 reads as a number.
		return fmt.Sprint(v)
	default:
		r.fail(name, v, "text")
		return ""
	}
}

func (r *reader) int(name string) int {
	switch v := r.fm[r.m.Field(name).Name].(type) {
	case nil:
		return 0
	case int:
		return v
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		if v == math.Trunc(v) {
			return int(v)
		}
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	r.fail(name, r.fm[r.m.Field(name).Name], "a whole number")
	return 0
}

// dateLayouts are the shapes a date field may be read in, whatever its
// format for writing.
var dateLayouts = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

func (r *reader) date(name string) time.Time {
	switch v := r.fm[r.m.Field(name).Name].(type) {
	case nil:
		return time.Time{}
	case Date:
		return v.Time
	case time.Time:
		return v
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
		for _, l := range dateLayouts {
			if t, err := time.ParseInLocation(l, v, time.Local); err == nil {
				return t
			}
		}
	}
	r.fail(name, r.fm[r.m.Field(name).Name], "a date")
	return time.Time{}
}

func (r *reader) list(name string) []string {
	switch v := r.fm[r.m.Field(name).Name].(type) {
	case nil:
		return nil
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			switch e := e.(type) {
			case string:
				out = append(out, e)
			case int, int64, uint64:
				out = append(out, fmt.Sprint(e))
			default:
				r.fail(name, v, "a list of text")
				return nil
			}
		}
		return out
	default:
		r.fail(name, v, "a list")
		return nil
	}
}
