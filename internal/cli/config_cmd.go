package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/alexsasharegan/zk/internal/config"
)

// configReport is what zk config prints, and its --json shape.
type configReport struct {
	Box        string `json:"box"`
	Spec       string `json:"spec"`
	Step       string `json:"step"`
	Reason     string `json:"reason"`
	UserConfig string `json:"user_config"`
}

func newConfigCmd(app *App) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print the box zk would act on, and why",
		Long: `Print the box zk would act on and which step chose it.

The order is: --box, then ZK_BOX, then the nearest zk.toml found walking up
from the working directory (stopping at the home directory), then the default
in the user config. The user config is $ZK_CONFIG, else
$XDG_CONFIG_HOME/zk/config.toml, else ~/.config/zk/config.toml.`,
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			env, err := app.Env()
			if err != nil {
				return err
			}
			user, err := config.LoadUser(env)
			if err != nil {
				return err
			}
			r, err := config.Resolve(env, user, app.box)
			if err != nil {
				return err
			}
			rep := configReport{Box: r.Name, Spec: r.Spec, Step: r.Step.String(), Reason: r.Reason, UserConfig: user.Path}
			out := cmd.OutOrStdout()
			if asJSON {
				return json.NewEncoder(out).Encode(rep)
			}
			name, userPath := rep.Box, rep.UserConfig
			if name == "" {
				name = "(not named in the user config)"
			}
			if userPath == "" {
				userPath = "(none)"
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "box\t%s\n", name)
			fmt.Fprintf(w, "spec\t%s\n", rep.Spec)
			fmt.Fprintf(w, "chosen\t%s\n", rep.Reason)
			fmt.Fprintf(w, "user config\t%s\n", userPath)
			return w.Flush()
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print one JSON object")
	return cmd
}
