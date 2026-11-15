package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexsasharegan/zk/internal/config"
)

// run runs zk in a fake home with the given environment variables.
func run(t *testing.T, home, cwd string, vars map[string]string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	app := &App{Out: &out, Err: &errb, Env: func() (config.Env, error) {
		return config.Env{Getenv: func(k string) string { return vars[k] }, Home: home, Cwd: cwd}, nil
	}}
	code = Run(app, args)
	return code, out.String(), errb.String()
}

func tempHome(t *testing.T) string {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return home
}

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBareZkPrintsHelp(t *testing.T) {
	home := tempHome(t)
	code, out, _ := run(t, home, home, nil)
	if code != ExitOK || !strings.Contains(out, "Available Commands") {
		t.Fatalf("code %d, out:\n%s", code, out)
	}
}

func TestCommandSet(t *testing.T) {
	home := tempHome(t)
	_, out, _ := run(t, home, home, nil, "--help")
	for _, c := range []string{"config", "version", "completion"} {
		if !strings.Contains(out, "\n  "+c+" ") {
			t.Errorf("help does not list %s:\n%s", c, out)
		}
	}
}

func TestVersion(t *testing.T) {
	home := tempHome(t)
	for _, args := range [][]string{{"version"}, {"--version"}} {
		code, out, _ := run(t, home, home, nil, args...)
		if code != ExitOK || out != "zk "+Version()+"\n" {
			t.Errorf("%v: code %d, out %q", args, code, out)
		}
	}
}

func TestCompletion(t *testing.T) {
	home := tempHome(t)
	code, out, _ := run(t, home, home, nil, "completion", "zsh")
	if code != ExitOK || !strings.Contains(out, "compdef") {
		t.Fatalf("code %d, out starts %.80q", code, out)
	}
}

func TestUsageErrorsExit2(t *testing.T) {
	home := tempHome(t)
	for _, args := range [][]string{{"nosuchcommand"}, {"config", "--nosuchflag"}, {"config", "extra"}, {"version", "extra"}} {
		code, _, errOut := run(t, home, home, nil, args...)
		if code != ExitUsage {
			t.Errorf("%v: code %d, want %d (stderr %q)", args, code, ExitUsage, errOut)
		}
	}
}

func TestConfigReportsDiscoveredBox(t *testing.T) {
	home := tempHome(t)
	spec := filepath.Join(home, "vault", "Slip Box", "zk.toml")
	write(t, spec, "name = \"Slip Box\"\n")
	cwd := filepath.Join(home, "vault", "Slip Box", "cards")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := run(t, home, cwd, nil, "config")
	if code != ExitOK {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	for _, want := range []string{spec, "found walking up from " + cwd, "(none)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}

	code, out, _ = run(t, home, cwd, nil, "config", "--json")
	var rep configReport
	if code != ExitOK || json.Unmarshal([]byte(out), &rep) != nil {
		t.Fatalf("code %d, out %q", code, out)
	}
	if rep.Spec != spec || rep.Step != "discovered" || strings.Count(out, "\n") != 1 {
		t.Errorf("json: %+v (raw %q)", rep, out)
	}
}

func TestConfigBoxFlagByName(t *testing.T) {
	home := tempHome(t)
	spec := filepath.Join(home, "notes", "box", "zk.toml")
	write(t, spec, "")
	write(t, filepath.Join(home, ".config", "zk", "config.toml"), "[boxes.notes]\nspec = \"~/notes/box\"\n")

	code, out, errOut := run(t, home, home, nil, "config", "--box", "notes", "--json")
	var rep configReport
	if code != ExitOK || json.Unmarshal([]byte(out), &rep) != nil {
		t.Fatalf("code %d, out %q, stderr %q", code, out, errOut)
	}
	if rep.Box != "notes" || rep.Step != "flag" || rep.Spec != spec {
		t.Errorf("got %+v", rep)
	}
}

func TestConfigNoBoxFails(t *testing.T) {
	home := tempHome(t)
	code, _, errOut := run(t, home, home, nil, "config")
	if code != ExitFail || !strings.Contains(errOut, "no box found") {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
}
