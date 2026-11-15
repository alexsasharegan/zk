// Package frontmatter reads and edits the YAML front matter of a Markdown
// note (ADR 261115-t9jd). Edits go through the YAML node tree, so keys keep
// their order and quoting, comments survive, and the body is kept byte for
// byte. It is the only package that imports the YAML library.
package frontmatter

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	yaml "go.yaml.in/yaml/v3"
)

// Plain is implemented by values written as a plain, unquoted scalar, such
// as a date (box.Date). Its result must be a valid plain YAML scalar.
type Plain interface {
	PlainScalar() string
}

// Doc is a parsed note: its front matter and its body.
type Doc struct {
	root *yaml.Node // a mapping node
	body []byte
	had  bool // the note had a front matter block
}

const fence = "---"

// Parse splits a note into front matter and body. A note without a leading
// "---" line has no front matter; its whole content is the body.
func Parse(data []byte) (*Doc, error) {
	d := &Doc{root: &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}}
	yamlPart, body, ok := split(data)
	if !ok {
		d.body = data
		return d, nil
	}
	d.had, d.body = true, body
	if len(bytes.TrimSpace(yamlPart)) == 0 {
		return d, nil
	}
	var n yaml.Node
	if err := yaml.Unmarshal(yamlPart, &n); err != nil {
		return nil, fmt.Errorf("front matter: %w", err)
	}
	if n.Kind != yaml.DocumentNode || len(n.Content) != 1 || n.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("front matter: not a set of key: value fields")
	}
	d.root = n.Content[0]
	return d, nil
}

// split returns the YAML between the opening and closing fence lines, and
// the body after the closing fence.
func split(data []byte) (yamlPart, body []byte, ok bool) {
	first, rest, found := cutLine(data)
	if !found || string(bytes.TrimRight(first, "\r")) != fence {
		return nil, nil, false
	}
	for pos := 0; ; {
		line, next, more := cutLine(rest[pos:])
		if string(bytes.TrimRight(line, "\r")) == fence {
			return rest[:pos], next, true
		}
		if !more {
			return nil, nil, false // an opening fence with no closing one is body text
		}
		pos += len(line) + 1
	}
}

func cutLine(b []byte) (line, rest []byte, found bool) {
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		return b[:i], b[i+1:], true
	}
	return b, nil, false
}

// Body returns the note's body, everything after the front matter.
func (d *Doc) Body() []byte { return d.body }

// SetBody replaces the note's body.
func (d *Doc) SetBody(b []byte) { d.body = b }

// Fields decodes the front matter into a map. Unquoted dates arrive as
// time.Time; everything else as the YAML library decodes it.
func (d *Doc) Fields() (map[string]any, error) {
	m := map[string]any{}
	if len(d.root.Content) == 0 {
		return m, nil
	}
	if err := d.root.Decode(&m); err != nil {
		return nil, fmt.Errorf("front matter: %w", err)
	}
	return m, nil
}

// Update decodes the fields, lets fn change the map, and writes the changes
// back into the node tree: a removed key is deleted, a changed value is
// replaced in place (a string keeps its old quoting), a new key is added at
// the end in sorted order, and an unchanged value is not touched.
func (d *Doc) Update(fn func(map[string]any)) error {
	fields, err := d.Fields()
	if err != nil {
		return err
	}
	old := make(map[string]any, len(fields))
	for k, v := range fields {
		old[k] = v
	}
	fn(fields)

	kept := d.root.Content[:0:0]
	for i := 0; i+1 < len(d.root.Content); i += 2 {
		k, v := d.root.Content[i], d.root.Content[i+1]
		nv, present := fields[k.Value]
		if !present {
			continue
		}
		if !same(old[k.Value], nv) {
			n, err := encode(nv)
			if err != nil {
				return fmt.Errorf("front matter %s: %w", k.Value, err)
			}
			keepStyle(v, n)
			v = n
		}
		kept = append(kept, k, v)
	}
	var added []string
	for k := range fields {
		if _, ok := old[k]; !ok {
			added = append(added, k)
		}
	}
	sort.Strings(added)
	for _, k := range added {
		n, err := encode(fields[k])
		if err != nil {
			return fmt.Errorf("front matter %s: %w", k, err)
		}
		kept = append(kept, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}, n)
	}
	d.root.Content = kept
	return nil
}

// same reports whether a new value would decode to the old one, so an
// unchanged field keeps its exact text.
func same(old, nv any) bool {
	n, err := encode(nv)
	if err != nil {
		return false
	}
	var decoded any
	if err := n.Decode(&decoded); err != nil {
		return false
	}
	return reflect.DeepEqual(old, decoded)
}

// keepStyle gives a replacement string scalar the quoting of the old one.
func keepStyle(old, n *yaml.Node) {
	if old.Kind != yaml.ScalarNode || n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
		return
	}
	switch old.Style {
	case yaml.DoubleQuotedStyle, yaml.SingleQuotedStyle:
		n.Style = old.Style
	}
}

func encode(v any) (*yaml.Node, error) {
	switch v := v.(type) {
	case Plain:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v.PlainScalar()}, nil
	case []string:
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, s := range v {
			e := &yaml.Node{}
			if err := e.Encode(s); err != nil {
				return nil, err
			}
			seq.Content = append(seq.Content, e)
		}
		return seq, nil
	}
	n := &yaml.Node{}
	if err := n.Encode(v); err != nil {
		return nil, err
	}
	return n, nil
}

// Bytes returns the note: the front matter between fences, then the body.
// A note that had no front matter and still has none is its body alone.
func (d *Doc) Bytes() ([]byte, error) {
	if len(d.root.Content) == 0 {
		if !d.had {
			return d.body, nil
		}
		return append([]byte(fence+"\n"+fence+"\n"), d.body...), nil
	}
	var b bytes.Buffer
	b.WriteString(fence + "\n")
	e := yaml.NewEncoder(&b)
	e.SetIndent(2)
	if err := e.Encode(d.root); err != nil {
		return nil, err
	}
	if err := e.Close(); err != nil {
		return nil, err
	}
	b.WriteString(fence + "\n")
	b.Write(d.body)
	return b.Bytes(), nil
}

// ReadFile parses the note at path.
func ReadFile(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return d, nil
}

// WriteFile writes the note to path through a temporary file and a rename,
// keeping an existing file's permissions.
func WriteFile(path string, d *Doc) error {
	data, err := d.Bytes()
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".zk-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
