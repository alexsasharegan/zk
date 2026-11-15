package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Step is the resolution step that chose a box.
type Step int

// The resolution order of ADR 261115-zd6s; the first step that matches wins.
const (
	StepFlag Step = iota + 1
	StepEnv
	StepDiscovered
	StepDefault
)

func (s Step) String() string {
	switch s {
	case StepFlag:
		return "flag"
	case StepEnv:
		return "env"
	case StepDiscovered:
		return "discovered"
	case StepDefault:
		return "default"
	}
	return "unknown"
}

// Resolution is the box a command acts on, and why.
type Resolution struct {
	// Name is the box's name in the user config; empty for a box found by
	// path or discovery that the user config does not name.
	Name string
	// Spec is the absolute path of the box's spec file.
	Spec string
	// Step is the resolution step that chose it.
	Step Step
	// Reason says in words why this box was chosen.
	Reason string
}

// ErrNoBox is returned when no step finds a box.
var ErrNoBox = errors.New("no box found")

// specHint is added to every "spec not found" error: Obsidian neither shows
// nor, by default, syncs a .toml file, so a box's spec can be missing on a
// device without any sign in the app (ADR 261115-0tp4).
const specHint = "Obsidian does not show .toml files, and Obsidian Sync copies them only when \"other file types\" is on, so the spec may not have reached this device"

// Resolve chooses the box: the --box value, then $ZK_BOX, then the nearest
// spec found walking up from the working directory (stopping at the home
// directory), then the user config's default. Nothing is merged: the user
// config only locates boxes.
func Resolve(env Env, user User, flag string) (Resolution, error) {
	if flag != "" {
		r, err := lookup(env, user, flag)
		if err != nil {
			return Resolution{}, fmt.Errorf("--box %s: %w", flag, err)
		}
		r.Step = StepFlag
		r.Reason = "chosen by --box " + flag
		return r, nil
	}
	if v := env.Getenv("ZK_BOX"); v != "" {
		r, err := lookup(env, user, v)
		if err != nil {
			return Resolution{}, fmt.Errorf("ZK_BOX=%s: %w", v, err)
		}
		r.Step = StepEnv
		r.Reason = "chosen by ZK_BOX=" + v
		return r, nil
	}
	spec, err := Discover(env.Cwd, env.Home)
	if err != nil {
		return Resolution{}, err
	}
	if spec != "" {
		return Resolution{
			Name:   nameFor(user, spec),
			Spec:   spec,
			Step:   StepDiscovered,
			Reason: "found walking up from " + env.Cwd,
		}, nil
	}
	if user.Default != "" {
		spec, err := specPath(user.Boxes[user.Default].Spec)
		if err != nil {
			return Resolution{}, fmt.Errorf("default box %q in %s: %w", user.Default, user.Path, err)
		}
		return Resolution{
			Name:   user.Default,
			Spec:   spec,
			Step:   StepDefault,
			Reason: "the default in " + user.Path,
		}, nil
	}
	return Resolution{}, fmt.Errorf("%w: pass --box NAME or --box PATH, set ZK_BOX, run zk inside a box (a folder with %s), or set default in the user config", ErrNoBox, SpecFile)
}

// Discover walks up from dir looking for a spec file. It checks home itself
// and stops there, or at the filesystem root when dir is outside home. It
// returns "" when there is none.
func Discover(dir, home string) (string, error) {
	dir = filepath.Clean(dir)
	home = filepath.Clean(home)
	for {
		p := filepath.Join(dir, SpecFile)
		info, err := os.Stat(p)
		switch {
		case err == nil && !info.IsDir():
			return p, nil
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			return "", fmt.Errorf("looking for a box: %w", err)
		}
		parent := filepath.Dir(dir)
		if dir == home || parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// lookup reads a --box or ZK_BOX value: a box name from the user config,
// else a path to a spec file or the folder that holds one.
func lookup(env Env, user User, v string) (Resolution, error) {
	if b, ok := user.Boxes[v]; ok {
		spec, err := specPath(b.Spec)
		if err != nil {
			return Resolution{}, err
		}
		return Resolution{Name: v, Spec: spec}, nil
	}
	p := Expand(v, env.Home)
	if !filepath.IsAbs(p) {
		p = filepath.Join(env.Cwd, p)
	}
	spec, err := specPath(p)
	if err != nil {
		if looksLikeName(v) && errors.Is(err, fs.ErrNotExist) {
			where := "there is no user config"
			if user.Path != "" {
				where = "the user config " + user.Path + " has no such box"
			}
			return Resolution{}, fmt.Errorf("no box named %q (%s) and no such path", v, where)
		}
		return Resolution{}, err
	}
	return Resolution{Name: nameFor(user, spec), Spec: spec}, nil
}

// specPath turns a spec file or its folder into the spec file's absolute
// path, and checks that it exists.
func specPath(p string) (string, error) {
	p, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", specMissing(p, err)
	}
	if info.IsDir() {
		p = filepath.Join(p, SpecFile)
		if _, err := os.Stat(p); err != nil {
			return "", specMissing(p, err)
		}
	}
	return p, nil
}

func specMissing(p string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no spec at %s (%s): %w", p, specHint, fs.ErrNotExist)
	}
	return fmt.Errorf("reading %s: %w", p, err)
}

// nameFor returns the user config's name for a spec path, or "". When two
// names point at one spec, the first in sorted order wins.
func nameFor(user User, spec string) string {
	names := make([]string, 0, len(user.Boxes))
	for name := range user.Boxes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if s, err := specPath(user.Boxes[name].Spec); err == nil && s == spec {
			return name
		}
	}
	return ""
}

func looksLikeName(v string) bool {
	return !strings.ContainsAny(v, `/\~.`)
}
