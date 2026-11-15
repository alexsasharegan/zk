// Package config finds and reads the user config, which names boxes and the
// default box, and resolves which box a command acts on (ADR 261115-zd6s).
//
// Nothing here reads the process environment directly: callers pass an Env,
// so tests can run every resolution step without touching the real home
// directory.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// SpecFile is the name of a box's spec file, in the box's managed folder.
const SpecFile = "zk.toml"

// Env is the part of the process environment that resolution depends on.
type Env struct {
	// Getenv looks up an environment variable; empty means unset.
	Getenv func(string) string
	// Home is the user's home directory.
	Home string
	// Cwd is the working directory.
	Cwd string
}

// OSEnv returns the real environment.
func OSEnv() (Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Env{}, fmt.Errorf("finding the home directory: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return Env{}, fmt.Errorf("finding the working directory: %w", err)
	}
	return Env{Getenv: os.Getenv, Home: home, Cwd: cwd}, nil
}

// BoxRef is one named box in the user config.
type BoxRef struct {
	// Spec is the path of the box's spec file, or of the folder that holds it.
	Spec string `toml:"spec"`
}

// User is the user config. It only locates boxes; everything about a box is
// in the box's own spec.
type User struct {
	// Path is the file the config was read from; empty when there is none.
	Path    string            `toml:"-"`
	Default string            `toml:"default"`
	Boxes   map[string]BoxRef `toml:"boxes"`
}

// UserPath returns the path of the user config, or "" when there is none.
// The order is $ZK_CONFIG, $XDG_CONFIG_HOME/zk/config.toml, then
// ~/.config/zk/config.toml. $ZK_CONFIG is explicit, so a missing file there
// is an error rather than a fall-through.
func UserPath(env Env) (string, error) {
	if p := env.Getenv("ZK_CONFIG"); p != "" {
		p = Expand(p, env.Home)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("ZK_CONFIG names %s, which cannot be read: %w", p, err)
		}
		return p, nil
	}
	var candidates []string
	if x := env.Getenv("XDG_CONFIG_HOME"); x != "" {
		candidates = append(candidates, filepath.Join(Expand(x, env.Home), "zk", "config.toml"))
	}
	candidates = append(candidates, filepath.Join(env.Home, ".config", "zk", "config.toml"))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("reading %s: %w", c, err)
		}
	}
	return "", nil
}

// LoadUser reads the user config. With no config file it returns an empty
// config and no error.
func LoadUser(env Env) (User, error) {
	path, err := UserPath(env)
	if err != nil || path == "" {
		return User{}, err
	}
	var u User
	md, err := toml.DecodeFile(path, &u)
	if err != nil {
		return User{}, fmt.Errorf("reading the user config %s: %w", path, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, len(undecoded))
		for i, k := range undecoded {
			keys[i] = k.String()
		}
		return User{}, fmt.Errorf("the user config %s has unknown keys: %s", path, strings.Join(keys, ", "))
	}
	u.Path = path
	for name, b := range u.Boxes {
		if b.Spec == "" {
			return User{}, fmt.Errorf("the user config %s: box %q has no spec", path, name)
		}
		b.Spec = Expand(b.Spec, env.Home)
		if !filepath.IsAbs(b.Spec) {
			b.Spec = filepath.Join(filepath.Dir(path), b.Spec)
		}
		u.Boxes[name] = b
	}
	if u.Default != "" {
		if _, ok := u.Boxes[u.Default]; !ok {
			return User{}, fmt.Errorf("the user config %s: default box %q is not defined under [boxes]", path, u.Default)
		}
	}
	return u, nil
}

// Expand replaces a leading ~ with the home directory.
func Expand(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}
