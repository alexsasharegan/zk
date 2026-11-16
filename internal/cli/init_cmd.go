package cli

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexsasharegan/zk/internal/box"
	"github.com/alexsasharegan/zk/internal/config"
)

func newInitCmd(app *App) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "init [folder]",
		Short: "Make a new box in a folder of an Obsidian vault",
		Long: `Make a new box in folder (the working directory when left out), creating the
folder if need be: zk.toml, "<name> Hub.md", "<name> Base.base", and the
cards and sources folders. The name is the folder's name unless --name says
otherwise. Then add the missing property types to .obsidian/types.json.

It writes nothing if the folder is not in a vault, is in a box or holds one,
or already holds one of those files. It never prompts.`,
		Example: `  zk init "Slip Box"
  zk init "3 AREAS/Slip Box" --name pkm`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MaximumNArgs(1)(cmd, args); err != nil {
				return UsageError{err}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := app.Env()
			if err != nil {
				return err
			}
			dir := env.Cwd
			if len(args) == 1 {
				dir = config.Expand(args[0], env.Home)
				if !filepath.IsAbs(dir) {
					dir = filepath.Join(env.Cwd, dir)
				}
			}
			res, err := box.Init(dir, name)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			s := res.Spec
			home := func(p string) string {
				if env.Home != "" && strings.HasPrefix(p, env.Home+string(filepath.Separator)) {
					return "~" + p[len(env.Home):]
				}
				return p
			}
			rel := func(p string) string {
				if r, err := filepath.Rel(s.Dir, p); err == nil {
					return r
				}
				return p
			}
			fmt.Fprintf(out, "made the box %q at %s\n", s.Name, home(s.Dir))
			for _, p := range res.Wrote {
				fmt.Fprintf(out, "  wrote %s\n", rel(p))
			}
			for _, p := range res.Made {
				if p != s.Dir {
					fmt.Fprintf(out, "  made %s/\n", rel(p))
				}
			}
			if len(res.Types) > 0 {
				fmt.Fprintf(out, "added %d property types to %s; reload Obsidian so it reads them\n", len(res.Types), box.TypesFile)
			}
			for _, c := range res.Conflicts {
				fmt.Fprintf(out, "warning: %s is %s in Obsidian, the box wants %s; zk check will report it\n", c.Name, c.Have, c.Want)
			}
			key := boxKey(s.Name)
			fmt.Fprintf(out, `
zk finds the box when run inside its folder. To use it from anywhere, name
it in the user config (zk config says where that is):

  [boxes.%s]
  spec = %q

and add default = %q there to use it when no box is found from the working
directory.
`, key, home(s.Path), key)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "the box's name, used for the hub and base files (default: the folder's name)")
	return cmd
}

var nonKey = regexp.MustCompile(`[^a-z0-9]+`)

// boxKey makes a box name into a bare TOML key for the user config.
func boxKey(name string) string {
	k := strings.Trim(nonKey.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if k == "" {
		return "box"
	}
	return k
}
