package box

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPropertyTypesDefault(t *testing.T) {
	got := DefaultMapping().PropertyTypes()
	want := map[string]string{
		"zk_class": "text", "zk_kind": "text", "zk_key": "text", "zk_level": "number",
		"zk_sort": "text", "zk_status": "text", "zk_created": "datetime", "zk_authored_by": "text",
		"zk_title": "text", "zk_author": "text", "zk_published": "date", "zk_provenance": "text",
		"zk_via": "text", "zk_form": "text", "zk_extracted": "multitext", "zk_series": "text",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestPropertyTypesFollowTheMapping(t *testing.T) {
	got := pkmMapping(t).PropertyTypes()
	for name, typ := range map[string]string{"type": "text", "created_at": "date", "published": "date", "zk_level": "number"} {
		if got[name] != typ {
			t.Errorf("%s is %q, want %q", name, got[name], typ)
		}
	}
	for _, name := range []string{"zk_class", "zk_created", "aliases", "tags"} {
		if _, ok := got[name]; ok {
			t.Errorf("%s should not be in the types", name)
		}
	}
}

func typesVault(t *testing.T, body string) string {
	t.Helper()
	p := vault(t, TypesFile, body)
	return filepath.Dir(filepath.Dir(p))
}

func readJSON(t *testing.T, p string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAddTypesAddsOnlyMissing(t *testing.T) {
	v := typesVault(t, `{"types": {"zk_created": "text", "zk_level": "number", "rating": "number"}, "other": [1, 2]}`)
	want := map[string]string{"zk_created": "datetime", "zk_level": "number", "zk_key": "text"}
	added, conflicts, err := AddTypes(v, want)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(added, []string{"zk_key"}) {
		t.Errorf("added %v", added)
	}
	if !reflect.DeepEqual(conflicts, []TypeConflict{{Name: "zk_created", Have: "text", Want: "datetime"}}) {
		t.Errorf("conflicts %v", conflicts)
	}
	got := readJSON(t, filepath.Join(v, TypesFile))
	types := got["types"].(map[string]any)
	if types["zk_created"] != "text" {
		t.Errorf("an existing type was changed: zk_created is %v", types["zk_created"])
	}
	if types["rating"] != "number" || types["zk_key"] != "text" {
		t.Errorf("types %v", types)
	}
	if !reflect.DeepEqual(got["other"], []any{1.0, 2.0}) {
		t.Errorf("another key was lost: %v", got)
	}

	// A second run has nothing to add and leaves the file as it is.
	before, _ := os.ReadFile(filepath.Join(v, TypesFile))
	added, _, err = AddTypes(v, want)
	if err != nil || len(added) != 0 {
		t.Fatalf("second run: added %v, err %v", added, err)
	}
	after, _ := os.ReadFile(filepath.Join(v, TypesFile))
	if string(before) != string(after) {
		t.Error("a run with nothing to add rewrote the file")
	}
}

func TestAddTypesCreatesTheFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	added, _, err := AddTypes(root, DefaultMapping().PropertyTypes())
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 16 {
		t.Errorf("added %d types, want 16", len(added))
	}
	types := readJSON(t, filepath.Join(root, TypesFile))["types"].(map[string]any)
	if types["zk_created"] != "datetime" || types["zk_level"] != "number" {
		t.Errorf("types %v", types)
	}
}

func TestAddTypesRejectsBadJSON(t *testing.T) {
	v := typesVault(t, `{"types": `)
	if _, _, err := AddTypes(v, map[string]string{"zk_key": "text"}); err == nil {
		t.Fatal("bad JSON should be an error")
	}
	data, _ := os.ReadFile(filepath.Join(v, TypesFile))
	if string(data) != `{"types": ` {
		t.Errorf("the file was changed: %q", data)
	}
}

func TestCheckTypes(t *testing.T) {
	p := vault(t, "box/zk.toml", "name = \"Box\"\n")
	spec, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Open(spec)
	if err != nil {
		t.Fatal(err)
	}
	findings := s.checkTypes(false)
	if len(findings) != 16 {
		t.Fatalf("no types.json: %d findings, want 16", len(findings))
	}
	for _, f := range findings {
		if !f.Warning || !f.Fixable || f.Path != TypesFile || f.Rule != "property-type" {
			t.Errorf("a missing type should be a fixable warning: %+v", f)
		}
	}
	for _, f := range s.checkTypes(true) {
		if f.Warning {
			t.Errorf("strict: a missing type should be an error: %+v", f)
		}
	}

	if _, _, err := AddTypes(spec.Vault, spec.Mapping.PropertyTypes()); err != nil {
		t.Fatal(err)
	}
	if f := s.checkTypes(true); len(f) != 0 {
		t.Errorf("after AddTypes: %v", f)
	}

	write := func(body string) {
		if err := os.WriteFile(filepath.Join(spec.Vault, TypesFile), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(filepath.Join(spec.Vault, TypesFile))
	write(strings.Replace(string(data), `"zk_created": "datetime"`, `"zk_created": "text"`, 1))
	findings = s.checkTypes(false)
	if len(findings) != 1 || findings[0].Warning || findings[0].Fixable || !strings.Contains(findings[0].Message, "zk_created is text in Obsidian, the spec wants datetime") {
		t.Errorf("a mismatch should be one error that is not fixable: %+v", findings)
	}

	write("not json")
	findings = s.checkTypes(false)
	if len(findings) != 1 || findings[0].Warning || !strings.Contains(findings[0].Message, "cannot read") {
		t.Errorf("bad JSON: %+v", findings)
	}
}
