// Package cli is the zk command tree (ADR 261115-sbpe).
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexsasharegan/zk/internal/box"
	"github.com/alexsasharegan/zk/internal/config"
	"github.com/alexsasharegan/zk/internal/open"
	"github.com/alexsasharegan/zk/internal/tui"
)

// Exit codes (ADR 261115-sbpe).
const (
	ExitOK    = 0
	ExitFail  = 1 // findings, or a failed operation
	ExitUsage = 2 // a usage error
)

// version is set at build time with -ldflags "-X .../internal/cli.version=v1.2.3".
var version = ""

// Version returns the build's version: the ldflags value, else the module
// version recorded by go install, else "dev".
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// UsageError marks an error as a usage error, so it exits with ExitUsage.
type UsageError struct{ Err error }

func (e UsageError) Error() string { return e.Err.Error() }
func (e UsageError) Unwrap() error { return e.Err }

// App holds what every command needs: output streams and the environment.
type App struct {
	Out, Err io.Writer
	// Env returns the environment; tests replace it.
	Env func() (config.Env, error)
	// Now is the clock; tests replace it.
	Now func() time.Time
	// Terminal reports whether stdin and stdout are both terminals.
	Terminal func() bool
	// ErrTerminal reports whether stderr is a terminal.
	ErrTerminal func() bool
	// In is read for a yes or no answer on a terminal.
	In io.Reader
	// Opener opens a new card in Obsidian or the editor.
	Opener open.Opener
	// NewUI runs the zk new terminal UI; tests replace it.
	NewUI func(tui.Input) (tui.Result, error)

	resolution config.Resolution // the box Open chose, and why

	box     string // --box
	noInput bool   // --no-input
}

// NewRoot builds the command tree.
func NewRoot(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:   "zk",
		Short: "A Zettelkasten slip box in an Obsidian vault",
		Long: `zk runs a slip box inside an Obsidian vault: cards with permanent keys,
the sources they were harvested from, and the hub note and base that show them.

With every argument given, zk never prompts, so scripts and agents can call it.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version(),
		Args:          noArgs,
		// Bare zk prints help. A root with no Run would also print help for
		// an unknown command and exit 0; with a Run, noArgs rejects it.
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	root.SetOut(app.Out)
	root.SetErr(app.Err)
	root.SetVersionTemplate("zk {{.Version}}\n")
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return UsageError{err} })
	root.PersistentFlags().StringVar(&app.box, "box", "", "the box to act on: a name from the user config, or a path to a box's spec or its folder")
	root.PersistentFlags().BoolVar(&app.noInput, "no-input", false, "never prompt, even on a terminal (also ZK_NO_INPUT=1)")

	root.AddCommand(newVersionCmd(), newConfigCmd(app),
		newNewCmd(app), newListCmd(app), newShowCmd(app), newRetitleCmd(app),
		newCheckCmd(app), newFixCmd(app), newSourceCmd(app))
	return root
}

// Interactive reports whether a command may prompt: stdin and stdout are
// terminals, and neither --no-input nor ZK_NO_INPUT=1 says otherwise
// (ADR 261115-czb2).
func (app *App) Interactive() bool {
	if app.noInput || app.Terminal == nil || !app.Terminal() {
		return false
	}
	env, err := app.Env()
	return err == nil && env.Getenv("ZK_NO_INPUT") != "1"
}

// Open resolves the box, loads its spec and reads its notes.
func (app *App) Open() (*box.Store, error) {
	env, err := app.Env()
	if err != nil {
		return nil, err
	}
	user, err := config.LoadUser(env)
	if err != nil {
		return nil, err
	}
	r, err := config.Resolve(env, user, app.box)
	if err != nil {
		return nil, err
	}
	app.resolution = r
	spec, err := box.Load(r.Spec)
	if err != nil {
		return nil, err
	}
	return box.Open(spec)
}

// noArgs is cobra.NoArgs, reported as a usage error.
func noArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return UsageError{err}
	}
	return nil
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "zk %s\n", Version())
			return err
		},
	}
}

// Execute runs zk with the process's arguments and returns the exit code.
func Execute() int {
	app := &App{Out: os.Stdout, Err: os.Stderr, Env: config.OSEnv, Now: time.Now, Terminal: func() bool {
		return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	}, ErrTerminal: func() bool {
		return term.IsTerminal(int(os.Stderr.Fd()))
	}, In: os.Stdin, Opener: open.System(), NewUI: func(in tui.Input) (tui.Result, error) {
		return tui.Run(in, os.Stdin, os.Stdout)
	}}
	return Run(app, os.Args[1:])
}

// Run runs zk with the given arguments and returns the exit code.
func Run(app *App, args []string) int {
	root := NewRoot(app)
	root.SetArgs(args)
	cmd, err := root.ExecuteC()
	code := app.report(err)
	app.noteBox(cmd)
	return code
}

// report prints a command's error and returns its exit code.
func (app *App) report(err error) int {
	if err == nil {
		return ExitOK
	}
	if errors.Is(err, errFindings) {
		return ExitFail // the findings are already printed
	}
	fmt.Fprintf(app.Err, "zk: %v\n", err)
	var usage UsageError
	var arg box.ArgError
	if errors.As(err, &usage) || errors.As(err, &arg) || isCobraUsage(err) {
		return ExitUsage
	}
	return ExitFail
}

// noteBox says on stderr which box a command acted on, when the box was not
// found from the working directory but chosen by ZK_BOX or the user config's
// default (ADR 261115-v1yz). It is the last line, so it sits above the prompt
// however long the output was. A box found walking up from here, or named with
// --box, needs no note. Only on a terminal, and never with --json.
func (app *App) noteBox(cmd *cobra.Command) {
	r := app.resolution
	if r.Step != config.StepEnv && r.Step != config.StepDefault {
		return
	}
	if app.ErrTerminal == nil || !app.ErrTerminal() {
		return
	}
	if cmd != nil {
		if f := cmd.Flags().Lookup("json"); f != nil && f.Value.String() == "true" {
			return
		}
	}
	line := "zk: box"
	if r.Name != "" {
		line += " " + r.Name
	}
	line += " at " + filepath.Dir(r.Spec) + " (" + r.Reason + ")"
	if env, err := app.Env(); err == nil && env.Home != "" {
		line = strings.ReplaceAll(line, env.Home+string(filepath.Separator), "~"+string(filepath.Separator))
	}
	fmt.Fprintln(app.Err, line)
}

// isCobraUsage recognises the errors Cobra returns for an unknown command,
// which do not pass through the flag error function.
func isCobraUsage(err error) bool {
	return strings.HasPrefix(err.Error(), "unknown command ")
}
