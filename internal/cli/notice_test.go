package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexsasharegan/zk/internal/config"
)

// noticeEnv is a home with a vault holding one box, named "notes" as the
// default in a user config.
type noticeEnv struct {
	home, box, config string
}

func newNoticeEnv(t *testing.T) noticeEnv {
	t.Helper()
	home := tempHome(t)
	e := noticeEnv{home: home, box: filepath.Join(home, "vault", "Slip Box"), config: filepath.Join(home, "zk.toml.conf")}
	write(t, filepath.Join(home, "vault", ".obsidian", "app.json"), "{}")
	write(t, filepath.Join(e.box, "zk.toml"), "name = \"Slip Box\"\n")
	for _, d := range []string{"cards", "sources"} {
		if err := os.MkdirAll(filepath.Join(e.box, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, e.config, "default = \"notes\"\n\n[boxes.notes]\nspec = \"~/vault/Slip Box\"\n")
	return e
}

// zk runs zk from cwd with stderr on a terminal or not.
func (e noticeEnv) zk(t *testing.T, cwd string, errTerminal bool, vars map[string]string, args ...string) (int, string, string) {
	t.Helper()
	all := map[string]string{"ZK_CONFIG": e.config}
	for k, v := range vars {
		all[k] = v
	}
	var out, errb bytes.Buffer
	app := &App{Out: &out, Err: &errb,
		Env: func() (config.Env, error) {
			return config.Env{Getenv: func(k string) string { return all[k] }, Home: e.home, Cwd: cwd}, nil
		},
		Now:         time.Now,
		Terminal:    func() bool { return false },
		ErrTerminal: func() bool { return errTerminal },
	}
	code := Run(app, args)
	return code, out.String(), errb.String()
}

func TestNoticeNamesTheDefaultBoxLast(t *testing.T) {
	e := newNoticeEnv(t)
	elsewhere := filepath.Join(e.home, "code")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errs := e.zk(t, elsewhere, true, nil, "list", "--tree")
	want := "zk: box notes at ~/vault/Slip Box (the default in ~/zk.toml.conf)\n"
	if code != ExitOK || errs != want {
		t.Fatalf("code %d, stderr %q, want %q", code, errs, want)
	}
	if !strings.HasPrefix(out, "Slip Box") {
		t.Errorf("stdout changed: %q", out)
	}
}

func TestNoticeNamesZKBox(t *testing.T) {
	e := newNoticeEnv(t)
	_, _, errs := e.zk(t, e.home, true, map[string]string{"ZK_BOX": "notes"}, "list")
	if want := "zk: box notes at ~/vault/Slip Box (chosen by ZK_BOX=notes)\n"; errs != want {
		t.Fatalf("stderr %q, want %q", errs, want)
	}
}

func TestNoticeComesAfterAnError(t *testing.T) {
	e := newNoticeEnv(t)
	code, _, errs := e.zk(t, e.home, true, nil, "new", "bogus", "Title", "--no-input")
	lines := strings.Split(strings.TrimSuffix(errs, "\n"), "\n")
	last := lines[len(lines)-1]
	if code != ExitUsage || !strings.Contains(errs, `"bogus" is not a kind`) || !strings.HasPrefix(last, "zk: box notes at ") {
		t.Fatalf("code %d, stderr %q", code, errs)
	}
}

func TestNoNotice(t *testing.T) {
	e := newNoticeEnv(t)
	cases := []struct {
		name        string
		cwd         string
		errTerminal bool
		args        []string
	}{
		{"found walking up from here", filepath.Join(e.box, "cards"), true, []string{"list"}},
		{"named with --box", e.home, true, []string{"list", "--box", "notes"}},
		{"stderr is not a terminal", e.home, false, []string{"list"}},
		{"--json", e.home, true, []string{"list", "--json"}},
		{"zk config, which says it already", e.home, true, []string{"config"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, _, errs := e.zk(t, c.cwd, c.errTerminal, nil, c.args...)
			if code != ExitOK || errs != "" {
				t.Fatalf("code %d, stderr %q", code, errs)
			}
		})
	}
}
