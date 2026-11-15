package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture is a fake home directory with an environment that points at it.
type fixture struct {
	t    *testing.T
	home string
	vars map[string]string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, home: home, vars: map[string]string{}}
}

func (f *fixture) env(cwd string) Env {
	return Env{Getenv: func(k string) string { return f.vars[k] }, Home: f.home, Cwd: cwd}
}

// write creates a file under home, with its folders.
func (f *fixture) write(rel, body string) string {
	f.t.Helper()
	p := filepath.Join(f.home, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return p
}

// dir creates a folder under home.
func (f *fixture) dir(rel string) string {
	f.t.Helper()
	p := filepath.Join(f.home, rel)
	if err := os.MkdirAll(p, 0o755); err != nil {
		f.t.Fatal(err)
	}
	return p
}

func TestUserPathOrder(t *testing.T) {
	f := newFixture(t)
	if p, err := UserPath(f.env(f.home)); err != nil || p != "" {
		t.Fatalf("no config: got %q, %v; want empty, nil", p, err)
	}

	home := f.write(".config/zk/config.toml", "")
	if p, _ := UserPath(f.env(f.home)); p != home {
		t.Fatalf("home fallback: got %q, want %q", p, home)
	}

	xdg := f.write("xdg/zk/config.toml", "")
	f.vars["XDG_CONFIG_HOME"] = filepath.Join(f.home, "xdg")
	if p, _ := UserPath(f.env(f.home)); p != xdg {
		t.Fatalf("XDG_CONFIG_HOME: got %q, want %q", p, xdg)
	}

	explicit := f.write("elsewhere/zk.conf", "")
	f.vars["ZK_CONFIG"] = "~/elsewhere/zk.conf"
	if p, _ := UserPath(f.env(f.home)); p != explicit {
		t.Fatalf("ZK_CONFIG: got %q, want %q", p, explicit)
	}

	f.vars["ZK_CONFIG"] = filepath.Join(f.home, "missing.toml")
	if _, err := UserPath(f.env(f.home)); err == nil {
		t.Fatal("ZK_CONFIG naming a missing file: want an error, not a fall-through")
	}
}

func TestUserPathXDGMissingFallsThrough(t *testing.T) {
	f := newFixture(t)
	home := f.write(".config/zk/config.toml", "")
	f.vars["XDG_CONFIG_HOME"] = filepath.Join(f.home, "xdg")
	if p, _ := UserPath(f.env(f.home)); p != home {
		t.Fatalf("got %q, want the home fallback %q", p, home)
	}
}

func TestLoadUser(t *testing.T) {
	f := newFixture(t)
	f.write(".config/zk/config.toml", `
default = "notes"

[boxes.notes]
spec = "~/notes/Slip Box/zk.toml"

[boxes.demo]
spec = "demo"
`)
	u, err := LoadUser(f.env(f.home))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := u.Boxes["notes"].Spec, filepath.Join(f.home, "notes", "Slip Box", "zk.toml"); got != want {
		t.Errorf("~ expansion: got %q, want %q", got, want)
	}
	if got, want := u.Boxes["demo"].Spec, filepath.Join(f.home, ".config", "zk", "demo"); got != want {
		t.Errorf("relative spec: got %q, want %q (relative to the config file)", got, want)
	}
	if u.Default != "notes" {
		t.Errorf("default: got %q", u.Default)
	}
}

func TestLoadUserRejects(t *testing.T) {
	cases := map[string]string{
		"unknown key":       "colour = \"blue\"\n",
		"undefined default": "default = \"nope\"\n",
		"box without spec":  "[boxes.empty]\n",
		"bad toml":          "default = \n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.write(".config/zk/config.toml", body)
			if _, err := LoadUser(f.env(f.home)); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

// box makes a box folder with a spec under home and returns the spec path.
func (f *fixture) box(rel string) string {
	return f.write(filepath.Join(rel, SpecFile), "name = \"test\"\n")
}

func TestResolveOrder(t *testing.T) {
	f := newFixture(t)
	named := f.box("named")
	byEnv := f.box("by-env")
	nearby := f.box("vault/box")
	deep := f.dir("vault/box/cards/sub")
	user := User{
		Path:    filepath.Join(f.home, ".config/zk/config.toml"),
		Default: "named",
		Boxes:   map[string]BoxRef{"named": {Spec: named}, "near": {Spec: filepath.Dir(nearby)}},
	}

	// 1. --box beats everything.
	f.vars["ZK_BOX"] = filepath.Dir(byEnv)
	r, err := Resolve(f.env(deep), user, "named")
	if err != nil || r.Step != StepFlag || r.Spec != named || r.Name != "named" {
		t.Fatalf("flag: got %+v, %v", r, err)
	}

	// 2. ZK_BOX beats discovery and the default.
	r, err = Resolve(f.env(deep), user, "")
	if err != nil || r.Step != StepEnv || r.Spec != byEnv {
		t.Fatalf("env: got %+v, %v", r, err)
	}

	// 3. Discovery beats the default, and names the box when the config does.
	delete(f.vars, "ZK_BOX")
	r, err = Resolve(f.env(deep), user, "")
	if err != nil || r.Step != StepDiscovered || r.Spec != nearby || r.Name != "near" {
		t.Fatalf("discovered: got %+v, %v", r, err)
	}
	if !strings.Contains(r.Reason, deep) {
		t.Errorf("discovered reason %q should name the working directory", r.Reason)
	}

	// 4. The default, outside any box.
	r, err = Resolve(f.env(f.dir("plain")), user, "")
	if err != nil || r.Step != StepDefault || r.Spec != named {
		t.Fatalf("default: got %+v, %v", r, err)
	}

	// 5. Nothing at all.
	_, err = Resolve(f.env(f.dir("plain")), User{}, "")
	if !errors.Is(err, ErrNoBox) {
		t.Fatalf("none: got %v, want ErrNoBox", err)
	}
}

func TestResolveFlagPath(t *testing.T) {
	f := newFixture(t)
	spec := f.box("vault/box")
	cwd := f.dir("vault")
	for _, v := range []string{"box", "./box", "box/zk.toml", spec, "~/vault/box"} {
		r, err := Resolve(f.env(cwd), User{}, v)
		if err != nil || r.Spec != spec || r.Name != "" {
			t.Errorf("--box %s: got %+v, %v", v, r, err)
		}
	}
}

func TestResolveFlagUnknownName(t *testing.T) {
	f := newFixture(t)
	_, err := Resolve(f.env(f.home), User{}, "nosuchbox")
	if err == nil || !strings.Contains(err.Error(), `no box named "nosuchbox"`) {
		t.Fatalf("got %v", err)
	}
}

func TestResolveMissingSpecHint(t *testing.T) {
	f := newFixture(t)
	dir := f.dir("vault/box") // a folder with no zk.toml
	user := User{Default: "gone", Boxes: map[string]BoxRef{"gone": {Spec: dir}}}
	_, err := Resolve(f.env(f.home), user, "")
	if err == nil || !strings.Contains(err.Error(), "Obsidian Sync") {
		t.Fatalf("got %v, want the sync hint", err)
	}
}

func TestDiscoverStopsAtHome(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// A spec above home must not be found.
	if err := os.WriteFile(filepath.Join(root, SpecFile), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	cwd := filepath.Join(home, "a", "b")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if p, err := Discover(cwd, home); err != nil || p != "" {
		t.Fatalf("got %q, %v; want nothing found", p, err)
	}

	// A spec in home itself is found.
	inHome := filepath.Join(home, SpecFile)
	if err := os.WriteFile(inHome, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if p, _ := Discover(cwd, home); p != inHome {
		t.Fatalf("got %q, want %q", p, inHome)
	}
}

func TestDiscoverIgnoresDirectoryNamedLikeSpec(t *testing.T) {
	f := newFixture(t)
	f.dir("vault/" + SpecFile)
	if p, err := Discover(filepath.Join(f.home, "vault"), f.home); err != nil || p != "" {
		t.Fatalf("got %q, %v", p, err)
	}
}
