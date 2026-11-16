// Package open opens a note in Obsidian or in the user's editor
// (ADR 261115-mjrm).
package open

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Where says how to open a note.
type Where string

// The places a new card can be opened.
const (
	Obsidian Where = "obsidian"
	Editor   Where = "editor"
	None     Where = "none"
)

// Parse reads an --open value.
func Parse(s string) (Where, error) {
	switch w := Where(s); w {
	case Obsidian, Editor, None:
		return w, nil
	}
	return "", fmt.Errorf("--open must be obsidian, editor or none, not %q", s)
}

// Runner runs a command. With attach, the command gets the terminal's
// stdin, stdout and stderr (an editor); without, it runs detached from them
// (a desktop opener).
type Runner func(name string, args []string, attach bool) error

// Opener opens notes. The zero value is not usable; use System.
type Opener struct {
	GOOS   string
	Getenv func(string) string
	Run    Runner
}

// System is the opener for this machine.
func System() Opener {
	return Opener{GOOS: runtime.GOOS, Getenv: os.Getenv, Run: runCommand}
}

func runCommand(name string, args []string, attach bool) error {
	cmd := exec.Command(name, args...)
	if attach {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("%s: %w: %s", name, err, msg)
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// ObsidianURL is the link that opens a file in Obsidian, launching it when
// it is not running.
func ObsidianURL(path string) string {
	return "obsidian://open?path=" + strings.ReplaceAll(url.QueryEscape(path), "+", "%20")
}

// Open opens path where asked. None does nothing.
func (o Opener) Open(w Where, path string) error {
	switch w {
	case Obsidian:
		return o.Obsidian(path)
	case Editor:
		return o.Editor(path)
	}
	return nil
}

// Obsidian hands the file's obsidian:// link to the system's opener. It
// fails when nothing handles the link, as when Obsidian is not installed.
// It cannot tell whether the file is in a vault Obsidian knows.
func (o Opener) Obsidian(path string) error {
	link := ObsidianURL(path)
	var name string
	var args []string
	switch o.GOOS {
	case "darwin":
		name, args = "open", []string{link}
	case "windows":
		name, args = "cmd", []string{"/c", "start", "", link}
	default:
		name, args = "xdg-open", []string{link}
	}
	if err := o.Run(name, args, false); err != nil {
		return fmt.Errorf("could not open it in Obsidian (is Obsidian installed?): %w", err)
	}
	return nil
}

// ErrNoEditor is returned when neither $VISUAL nor $EDITOR is set.
var ErrNoEditor = errors.New("neither $VISUAL nor $EDITOR is set")

// EditorName is the editor command: $VISUAL, else $EDITOR, else "".
func (o Opener) EditorName() string {
	if v := o.Getenv("VISUAL"); v != "" {
		return v
	}
	return o.Getenv("EDITOR")
}

// Editor runs $VISUAL, else $EDITOR, on the file, attached to the terminal.
// The variable may hold arguments ("code --wait").
func (o Opener) Editor(path string) error {
	fields := strings.Fields(o.EditorName())
	if len(fields) == 0 {
		return ErrNoEditor
	}
	if err := o.Run(fields[0], append(fields[1:], path), true); err != nil {
		return fmt.Errorf("could not open it in %s: %w", fields[0], err)
	}
	return nil
}
