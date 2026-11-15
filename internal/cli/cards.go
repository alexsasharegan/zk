package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/alexsasharegan/zk/internal/box"
)

// usagef makes a usage error.
func usagef(format string, a ...any) error { return UsageError{fmt.Errorf(format, a...)} }

// exactArgs and minArgs are cobra's validators, reported as usage errors.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return UsageError{err}
		}
		return nil
	}
}

func minArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(n)(cmd, args); err != nil {
			return UsageError{err}
		}
		return nil
	}
}

// jsonLine prints one JSON object on its own line.
func jsonLine(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// kindList is the box's kinds with their descriptions, for help and errors.
func kindList(s *box.Store) string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	for _, name := range s.Spec.KindNames() {
		k := s.Spec.Kinds[name]
		rule := ""
		switch {
		case k.NeedsSource:
			rule = " (links its source page)"
		case k.NeedsLink:
			rule = " (links at least one card)"
		}
		fmt.Fprintf(w, "  %s\t%s%s\n", name, k.Description, rule)
	}
	w.Flush()
	return b.String()
}

func newNewCmd(app *App) *cobra.Command {
	var under string
	var dryRun, agent, asJSON bool
	cmd := &cobra.Command{
		Use:   "new <kind> <title...>",
		Short: "Make the next card: a new root, or the next child of --under KEY",
		Long: `Make the next card. With --under KEY it is KEY's next child (6 gives 6a, 6b;
6a gives 6a1, 6a2); without, the next root.

The kinds come from the box's spec; zk config lists them, and a call without a
kind prints them with what each means. With every argument given, zk new never
prompts.`,
		Example: `  zk new idea Taste is deletion
  zk new claim "Judges decide" --under 5
  zk new idea Another thought --under 6a --dry-run`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			if len(args) < 2 {
				what := "a kind and a title"
				if len(args) == 1 {
					what = "a title"
				}
				fmt.Fprintf(app.Err, "kinds in this box:\n%s", kindList(s))
				if app.Interactive() {
					return usagef("zk new needs %s (the terminal UI that asks for them is not built yet)", what)
				}
				return usagef("zk new needs %s", what)
			}
			title, err := box.CleanTitle(args[1:])
			if err != nil {
				return err
			}
			c := box.NewCard{Kind: args[0], Title: title, Under: under, Agent: agent, Now: app.Now()}
			if _, ok := s.Spec.Kinds[c.Kind]; !ok {
				fmt.Fprintf(app.Err, "kinds in this box:\n%s", kindList(s))
			}
			var k, p string
			if dryRun {
				k, p, err = s.PlanCard(c)
			} else {
				k, p, err = s.MakeCard(c)
			}
			if err != nil {
				return err
			}
			if asJSON {
				return jsonLine(cmd.OutOrStdout(), map[string]any{"key": k, "path": s.Rel(p), "dry_run": dryRun})
			}
			prefix := ""
			if dryRun {
				prefix = "would make "
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), prefix+s.Rel(p))
			return err
		},
	}
	cmd.Flags().StringVar(&under, "under", "", "make it the next child of this card")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the card it would make, and write nothing")
	cmd.Flags().BoolVar(&agent, "agent", false, "an agent made this card (sets authored_by)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print one JSON object")
	return cmd
}

func newListCmd(app *App) *cobra.Command {
	var f box.Filter
	var tree, asJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List cards depth first",
		Args:    noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			if f.Status != "" && !contains(box.CardStatuses, f.Status) {
				return usagef("--status must be one of %s", strings.Join(box.CardStatuses, ", "))
			}
			rows, err := s.List(f)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, r := range rows {
				if asJSON {
					if err := jsonLine(out, r); err != nil {
						return err
					}
					continue
				}
				k := r.Key
				if tree {
					k = strings.Repeat("  ", r.Level-1) + k
				}
				fmt.Fprintf(out, "%-10s %-9s %-9s %s\n", k, r.Status, r.Kind, r.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&f.Status, "status", "", "only cards with this status (raw, proposed, linked)")
	cmd.Flags().StringVar(&f.Kind, "kind", "", "only cards of this kind")
	cmd.Flags().StringVar(&f.Under, "under", "", "only the descendants of this card")
	cmd.Flags().BoolVar(&tree, "tree", false, "indent by level")
	cmd.Flags().BoolVar(&asJSON, "json", false, "one JSON object per card")
	return cmd
}

func newShowCmd(app *App) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show KEY",
		Short: "Explain a key, and list what the card links to and what links to it",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			info, err := s.Show(args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				return jsonLine(out, info)
			}
			dash := func(v string) string {
				if v == "" {
					return "-"
				}
				return v
			}
			parentExists := "-"
			if info.ParentExists != nil {
				parentExists = fmt.Sprint(*info.ParentExists)
			}
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "key\t%s\n", info.Key)
			fmt.Fprintf(w, "title\t%s\n", dash(info.Title))
			fmt.Fprintf(w, "level\t%d\n", info.Level)
			fmt.Fprintf(w, "sort\t%s\n", info.Sort)
			fmt.Fprintf(w, "tag\t%s\n", info.Tag)
			fmt.Fprintf(w, "file\t%s\n", info.File)
			fmt.Fprintf(w, "exists\t%v\n", info.Exists)
			fmt.Fprintf(w, "parent\t%s\n", dash(info.Parent))
			fmt.Fprintf(w, "parent_exists\t%s\n", parentExists)
			fmt.Fprintf(w, "next_child\t%s\n", dash(info.NextChild))
			if err := w.Flush(); err != nil {
				return err
			}
			if info.Exists {
				if len(info.Links) == 0 {
					fmt.Fprintln(out, "links\tnone")
				}
				w = tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
				for _, l := range info.Links {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", l.Direction, l.Target, l.Kind, l.Via)
				}
				return w.Flush()
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print one JSON object")
	return cmd
}

func newRetitleCmd(app *App) *cobra.Command {
	var links, asJSON bool
	cmd := &cobra.Command{
		Use:   "retitle KEY <title...>",
		Short: "Change a card's title: its H1 and nickname together",
		Long:  "Change a card's title: the # H1 and aliases[0] together. It refuses a nickname another card has.",
		Args:  minArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			title, err := box.CleanTitle(args[1:])
			if err != nil {
				return err
			}
			old, rewrote, files, err := s.Retitle(args[0], title, links)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				return jsonLine(out, map[string]any{"key": args[0], "old": old, "new": title, "links_rewritten": rewrote})
			}
			msg := fmt.Sprintf("retitled %s-%s: %q -> %q", s.Spec.KeyPrefix, args[0], old, title)
			if links {
				msg += fmt.Sprintf("; rewrote %d link(s) in %d file(s)", rewrote, files)
			}
			_, err = fmt.Fprintln(out, msg)
			return err
		},
	}
	cmd.Flags().BoolVar(&links, "links", false, `also rewrite old "[[zk-K|(K) Old title]]" links in cards and sources`)
	cmd.Flags().BoolVar(&asJSON, "json", false, "print one JSON object")
	return cmd
}

// errFindings makes zk check exit 1 without printing anything more.
var errFindings = errors.New("findings")

func newCheckCmd(app *App) *cobra.Command {
	var fix, strict, asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check cards and sources against the box's spec; exits 1 on an error",
		Long: `Check cards and sources against the box's spec. Exits 1 on an error; warnings
(a duplicate nickname, a skipped key) do not fail unless --strict.

With --fix, first write the derived fields (key, level, sort, the key's tag,
and aliases[0] from the H1), then check.`,
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if fix {
				changed, err := s.Fix(nil)
				if err != nil {
					return err
				}
				for _, k := range changed {
					if asJSON {
						jsonLine(out, map[string]any{"fixed": s.Rel(s.CardPath(k)), "key": k})
					} else {
						fmt.Fprintf(out, "fixed %s\n", s.Rel(s.CardPath(k)))
					}
				}
			}
			return report(app, s, s.Check(strict), asJSON)
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "write the derived fields first")
	cmd.Flags().BoolVar(&strict, "strict", false, "a duplicate nickname or a skipped key is an error, not a warning")
	cmd.Flags().BoolVar(&asJSON, "json", false, "one JSON object per finding")
	return cmd
}

func report(app *App, s *box.Store, findings []box.Finding, asJSON bool) error {
	out := app.Out
	errs := 0
	var fixable []string
	seen := map[string]bool{}
	for _, f := range findings {
		if !f.Warning {
			errs++
		}
		if f.Fixable && !f.Warning && f.Key != "" && !seen[f.Key] {
			seen[f.Key] = true
			fixable = append(fixable, f.Key)
		}
		if asJSON {
			if err := jsonLine(out, f); err != nil {
				return err
			}
			continue
		}
		prefix := ""
		if f.Warning {
			prefix = "warning: "
		}
		fmt.Fprintf(out, "%s%s [%s] %s\n", prefix, f.Path, f.Rule, f.Message)
	}
	if !asJSON {
		if len(findings) == 0 {
			fmt.Fprintln(out, "zk check: nothing found")
		}
		if len(fixable) > 0 {
			hint := "zk fix"
			if len(fixable) <= 12 {
				hint += " " + strings.Join(fixable, " ")
			}
			fmt.Fprintf(out, "fixable findings in %d card(s): run `%s`\n", len(fixable), hint)
		}
	}
	if errs > 0 {
		return errFindings
	}
	return nil
}

func newFixCmd(app *App) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "fix [KEY...]",
		Short: "Write the derived fields of cards (all, or the keys given); zk check --fix for every card",
		Long: `Write the derived fields of the cards given, or of every card: the class when
missing, key, level and sort from the file name, the key's tag, and aliases[0]
from the H1. It finishes a card made by hand from a template.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			changed, err := s.Fix(args)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, k := range changed {
				if asJSON {
					jsonLine(out, map[string]any{"fixed": s.Rel(s.CardPath(k)), "key": k})
				} else {
					fmt.Fprintf(out, "fixed %s\n", s.Rel(s.CardPath(k)))
				}
			}
			if len(changed) == 0 && !asJSON {
				fmt.Fprintln(out, "zk fix: nothing to fix")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "one JSON object per fixed card")
	return cmd
}

func newSourceCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Make source pages and mark them extracted",
		Args:  noArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}

	var src box.NewSource
	var newJSON bool
	newCmd := &cobra.Command{
		Use:   "new <slug>",
		Short: "Make a source page",
		Example: `  zk source new huntley-libraries --provenance https://example.com/libraries --form article
  zk source new physical-box --provenance "Zettelkasten #1 (physical box)" --form other`,
		Args: exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			src.Slug, src.Now = args[0], app.Now()
			p, err := s.MakeSource(src)
			if err != nil {
				return err
			}
			if newJSON {
				return jsonLine(cmd.OutOrStdout(), map[string]any{"slug": src.Slug, "path": s.Rel(p)})
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), s.Rel(p))
			return err
		},
	}
	newCmd.Flags().StringVar(&src.Provenance, "provenance", "", "where it comes from: a URL, a [[link]] to a note, or a description (required)")
	newCmd.Flags().StringVar(&src.Title, "title", "", "its title (default: from the slug)")
	newCmd.Flags().StringVar(&src.Author, "author", "", "who made it")
	newCmd.Flags().StringVar(&src.Via, "via", "", "how it reached you")
	newCmd.Flags().StringVar(&src.Form, "form", "article", "one of "+strings.Join(box.Forms, ", "))
	newCmd.Flags().BoolVar(&newJSON, "json", false, "print one JSON object")

	var doneJSON bool
	doneCmd := &cobra.Command{
		Use:   "done <slug> KEY...",
		Short: "Mark a source extracted, with the cards it made (added to any it lists)",
		Args:  minArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			merged, err := s.SourceDone(args[0], args[1:])
			if err != nil {
				return err
			}
			if doneJSON {
				return jsonLine(cmd.OutOrStdout(), map[string]any{"slug": args[0], "extracted": merged})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s: extracted [%s]\n", args[0], strings.Join(merged, ", "))
			return err
		},
	}
	doneCmd.Flags().BoolVar(&doneJSON, "json", false, "print one JSON object")

	cmd.AddCommand(newCmd, doneCmd)
	return cmd
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
