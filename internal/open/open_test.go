package open

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type call struct {
	name   string
	args   []string
	attach bool
}

func fake(goos string, env map[string]string, fail error) (Opener, *[]call) {
	var calls []call
	return Opener{
		GOOS:   goos,
		Getenv: func(k string) string { return env[k] },
		Run: func(name string, args []string, attach bool) error {
			calls = append(calls, call{name, args, attach})
			return fail
		},
	}, &calls
}

func TestObsidianURL(t *testing.T) {
	got := ObsidianURL("/Users/a/My Vault/Slip Box/cards/zk-2a.md")
	want := "obsidian://open?path=%2FUsers%2Fa%2FMy%20Vault%2FSlip%20Box%2Fcards%2Fzk-2a.md"
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestObsidianPerPlatform(t *testing.T) {
	link := ObsidianURL("/v/zk-1.md")
	for goos, want := range map[string]call{
		"darwin":  {"open", []string{link}, false},
		"linux":   {"xdg-open", []string{link}, false},
		"windows": {"cmd", []string{"/c", "start", "", link}, false},
	} {
		o, calls := fake(goos, nil, nil)
		if err := o.Open(Obsidian, "/v/zk-1.md"); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(*calls, []call{want}) {
			t.Errorf("%s: %+v", goos, *calls)
		}
	}
}

func TestObsidianFailureSaysWhy(t *testing.T) {
	o, _ := fake("darwin", nil, errors.New("exit status 1: No application knows how to open URL"))
	err := o.Open(Obsidian, "/v/zk-1.md")
	if err == nil || !strings.Contains(err.Error(), "is Obsidian installed?") {
		t.Fatalf("got %v", err)
	}
}

func TestEditor(t *testing.T) {
	o, calls := fake("darwin", map[string]string{"EDITOR": "nvim"}, nil)
	if err := o.Open(Editor, "/v/zk-1.md"); err != nil {
		t.Fatal(err)
	}
	if want := []call{{"nvim", []string{"/v/zk-1.md"}, true}}; !reflect.DeepEqual(*calls, want) {
		t.Errorf("got %+v", *calls)
	}

	o, calls = fake("darwin", map[string]string{"EDITOR": "nvim", "VISUAL": "code --wait"}, nil)
	o.Open(Editor, "/v/zk-1.md")
	if want := []call{{"code", []string{"--wait", "/v/zk-1.md"}, true}}; !reflect.DeepEqual(*calls, want) {
		t.Errorf("VISUAL wins, with its arguments: %+v", *calls)
	}

	o, calls = fake("darwin", nil, nil)
	if err := o.Open(Editor, "/v/zk-1.md"); !errors.Is(err, ErrNoEditor) || len(*calls) != 0 {
		t.Errorf("no editor set: %v, %+v", err, *calls)
	}
}

func TestNoneAndParse(t *testing.T) {
	o, calls := fake("darwin", nil, nil)
	if err := o.Open(None, "/v/zk-1.md"); err != nil || len(*calls) != 0 {
		t.Errorf("none should do nothing: %v %+v", err, *calls)
	}
	for _, s := range []string{"obsidian", "editor", "none"} {
		if w, err := Parse(s); err != nil || string(w) != s {
			t.Errorf("Parse(%s): %v %v", s, w, err)
		}
	}
	if _, err := Parse("vim"); err == nil {
		t.Error("Parse(vim) should fail")
	}
}
