package box

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func emptyVault(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".obsidian"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestInitMakesACleanBox(t *testing.T) {
	v := emptyVault(t)
	dir := filepath.Join(v, "3 AREAS", "Slip Box")
	res, err := Init(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Spec.Name != "Slip Box" || res.Spec.Vault != v {
		t.Errorf("spec %+v", res.Spec)
	}
	for _, f := range []string{"zk.toml", "Slip Box Hub.md", "Slip Box Base.base"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	for _, d := range []string{"cards", "sources"} {
		if info, err := os.Stat(filepath.Join(dir, d)); err != nil || !info.IsDir() {
			t.Errorf("%s is not a folder: %v", d, err)
		}
	}
	if len(res.Types) != 16 || len(res.Conflicts) != 0 {
		t.Errorf("types %v, conflicts %v", res.Types, res.Conflicts)
	}
	s, err := Open(res.Spec)
	if err != nil {
		t.Fatal(err)
	}
	if f := s.Check(true); len(f) != 0 {
		t.Errorf("a new box should check clean: %+v", f)
	}
}

func TestInitNameAndExistingFolders(t *testing.T) {
	v := emptyVault(t)
	dir := filepath.Join(v, "notes")
	note := filepath.Join(dir, "cards", "kept.md")
	if err := os.MkdirAll(filepath.Dir(note), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Init(dir, "pkm")
	if err != nil {
		t.Fatal(err)
	}
	if res.Spec.Name != "pkm" {
		t.Errorf("name %q", res.Spec.Name)
	}
	if _, err := os.Stat(filepath.Join(dir, "pkm Hub.md")); err != nil {
		t.Error(err)
	}
	if data, _ := os.ReadFile(note); string(data) != "mine\n" {
		t.Error("a note in the existing cards folder was touched")
	}
	if len(res.Made) != 1 || res.Made[0] != filepath.Join(dir, "sources") {
		t.Errorf("made %v, want only sources", res.Made)
	}
}

func TestInitKeepsTypesThatAreSet(t *testing.T) {
	v := emptyVault(t)
	if err := os.WriteFile(filepath.Join(v, TypesFile), []byte(`{"types": {"zk_created": "text"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Init(filepath.Join(v, "box"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0].Name != "zk_created" || len(res.Types) != 15 {
		t.Errorf("types %v, conflicts %v", res.Types, res.Conflicts)
	}
	if !strings.Contains(string(mustRead(t, filepath.Join(v, TypesFile))), `"zk_created": "text"`) {
		t.Error("zk init changed a type that was set")
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInitRefuses(t *testing.T) {
	v := emptyVault(t)
	if _, err := Init(filepath.Join(v, "box"), ""); err != nil {
		t.Fatal(err)
	}
	hubOnly := filepath.Join(v, "other")
	if err := os.MkdirAll(hubOnly, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hubOnly, "other Hub.md"), []byte("# mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	notVault, _ := filepath.EvalSymlinks(t.TempDir())

	cases := []struct {
		dir, name, want string
	}{
		{filepath.Join(v, "box"), "", "already a box"},
		{filepath.Join(v, "box", "cards", "inner"), "", "already a box"},
		{v, "", "holds a box already"},
		{hubOnly, "", "never overwrites"},
		{filepath.Join(notVault, "box"), "", "not inside an Obsidian vault"},
		{filepath.Join(v, "x"), "a/b", "cannot use"},
		{filepath.Join(v, "x"), "Notes [old]", "cannot use"},
		{filepath.Join(v, "x"), ".hidden", "dot"},
		{filepath.Join(v, "x"), " pad", "space"},
	}
	for _, c := range cases {
		_, err := Init(c.dir, c.name)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Init(%s, %q): %v, want %q", c.dir, c.name, err, c.want)
		}
	}
	if data := mustRead(t, filepath.Join(hubOnly, "other Hub.md")); string(data) != "# mine\n" {
		t.Error("a refused init changed a file")
	}
	for _, p := range []string{filepath.Join(hubOnly, "zk.toml"), filepath.Join(v, "x"), filepath.Join(notVault, "box")} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("a refused init wrote %s", p)
		}
	}
}
