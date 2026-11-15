// Package cli is the zk command tree (ADR 261115-sbpe).
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexsasharegan/zk/internal/config"
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

	box string // --box
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

	root.AddCommand(newVersionCmd(), newConfigCmd(app))
	return root
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
	app := &App{Out: os.Stdout, Err: os.Stderr, Env: config.OSEnv}
	return Run(app, os.Args[1:])
}

// Run runs zk with the given arguments and returns the exit code.
func Run(app *App, args []string) int {
	root := NewRoot(app)
	root.SetArgs(args)
	err := root.Execute()
	if err == nil {
		return ExitOK
	}
	fmt.Fprintf(app.Err, "zk: %v\n", err)
	var usage UsageError
	if errors.As(err, &usage) || isCobraUsage(err) {
		return ExitUsage
	}
	return ExitFail
}

// isCobraUsage recognises the errors Cobra returns for an unknown command,
// which do not pass through the flag error function.
func isCobraUsage(err error) bool {
	return strings.HasPrefix(err.Error(), "unknown command ")
}
