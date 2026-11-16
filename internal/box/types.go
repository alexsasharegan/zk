package box

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// TypesFile is where Obsidian keeps a vault's property types, relative to
// the vault: {"types": {"<property>": "<type>"}}. One type per property name
// for the whole vault (ADR 261115-2a31).
const TypesFile = ".obsidian/types.json"

// Obsidian's property types that zk's fields use.
const (
	TypeText      = "text"
	TypeMultitext = "multitext"
	TypeNumber    = "number"
	TypeDate      = "date"
	TypeDatetime  = "datetime"
)

// PropertyTypes returns the Obsidian type each field the mapping writes
// should have, by vault field name: a date field by its format, level a
// number, extracted a list of text, and the rest text. aliases and tags are
// Obsidian's and never appear.
func (m Mapping) PropertyTypes() map[string]string {
	out := make(map[string]string, len(canonical))
	for name, cf := range canonical {
		f := m.fields[name]
		t := TypeText
		switch {
		case cf.date && f.Format == FormatDate:
			t = TypeDate
		case cf.date:
			t = TypeDatetime
		case name == "level":
			t = TypeNumber
		case name == "extracted":
			t = TypeMultitext
		}
		out[f.Name] = t
	}
	return out
}

// TypeConflict is a property whose type in the vault is not the one wanted.
type TypeConflict struct {
	Name string
	Have string
	Want string
}

// Types is a vault's types.json as read: the types, and the file's other
// top-level keys, kept as they are for writing back.
type Types struct {
	Types map[string]string
	raw   map[string]json.RawMessage // top-level keys
	types map[string]json.RawMessage // entries, including ones that are not strings
}

// ReadTypes reads the vault's types.json. A missing file has no entries.
func ReadTypes(vault string) (*Types, error) {
	t := &Types{Types: map[string]string{}, raw: map[string]json.RawMessage{}, types: map[string]json.RawMessage{}}
	data, err := os.ReadFile(filepath.Join(vault, TypesFile))
	if errors.Is(err, fs.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return t, nil
	}
	if err := json.Unmarshal(data, &t.raw); err != nil {
		return nil, fmt.Errorf("%s: %w", TypesFile, err)
	}
	if rt, ok := t.raw["types"]; ok {
		if err := json.Unmarshal(rt, &t.types); err != nil {
			return nil, fmt.Errorf("%s: \"types\" is not an object: %w", TypesFile, err)
		}
	}
	for name, v := range t.types {
		var s string
		if json.Unmarshal(v, &s) == nil {
			t.Types[name] = s
		} else {
			t.Types[name] = string(v)
		}
	}
	if t.types == nil {
		t.types = map[string]json.RawMessage{}
	}
	return t, nil
}

// Compare returns the wanted properties that have no entry, sorted, and the
// ones whose entry is another type.
func (t *Types) Compare(want map[string]string) (missing []string, conflicts []TypeConflict) {
	for _, name := range sortedKeys(want) {
		have, ok := t.Types[name]
		switch {
		case !ok:
			missing = append(missing, name)
		case have != want[name]:
			conflicts = append(conflicts, TypeConflict{Name: name, Have: have, Want: want[name]})
		}
	}
	return missing, conflicts
}

// AddTypes adds to the vault's types.json the wanted types it has no entry
// for, creating the file if need be. It never changes an entry that is
// there, whatever its type, and keeps the file's other entries and keys. It
// returns the properties it added, sorted, and the ones set to another type.
// Obsidian may write its own copy over the file while it runs, so a caller
// should tell the user to reload it.
func AddTypes(vault string, want map[string]string) (added []string, conflicts []TypeConflict, err error) {
	t, err := ReadTypes(vault)
	if err != nil {
		return nil, nil, err
	}
	missing, conflicts := t.Compare(want)
	if len(missing) == 0 {
		return nil, conflicts, nil
	}
	for _, name := range missing {
		v, _ := json.Marshal(want[name])
		t.types[name] = v
	}
	rt, err := json.Marshal(t.types)
	if err != nil {
		return nil, nil, err
	}
	t.raw["types"] = rt
	data, err := json.MarshalIndent(t.raw, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := writeAtomic(filepath.Join(vault, TypesFile), append(data, '\n')); err != nil {
		return nil, nil, err
	}
	for _, name := range missing {
		t.Types[name] = want[name]
	}
	return missing, conflicts, nil
}

// writeAtomic writes a file through a temporary file in the same folder, so
// a reader never sees half of it.
func writeAtomic(p string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(p); err == nil {
		mode = info.Mode().Perm()
	}
	f, err := os.CreateTemp(filepath.Dir(p), "."+filepath.Base(p)+".*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// checkTypes reports the box's fields whose Obsidian type is missing (a
// warning unless strict) or another type (an error).
func (s *Store) checkTypes(strict bool) []Finding {
	t, err := ReadTypes(s.Spec.Vault)
	if err != nil {
		return []Finding{{Path: TypesFile, Rule: "property-type", Message: fmt.Sprintf("cannot read the property types: %v", err)}}
	}
	want := s.Spec.Mapping.PropertyTypes()
	missing, conflicts := t.Compare(want)
	var out []Finding
	for _, c := range conflicts {
		out = append(out, Finding{Path: TypesFile, Rule: "property-type", Message: fmt.Sprintf(
			"%s is %s in Obsidian, the spec wants %s: change it in Obsidian's property menu (other notes may use it), or map the field to another name in zk.toml",
			c.Name, c.Have, c.Want)})
	}
	for _, name := range missing {
		out = append(out, Finding{Path: TypesFile, Rule: "property-type", Fixable: true, Warning: !strict, Message: fmt.Sprintf(
			"%s has no type, so Obsidian guesses one; want %s", name, want[name])})
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
