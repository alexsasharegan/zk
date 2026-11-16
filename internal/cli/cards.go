package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/alexsasharegan/zk/internal/box"
	"github.com/alexsasharegan/zk/internal/open"
	"github.com/alexsasharegan/zk/internal/tree"
	"github.com/alexsasharegan/zk/internal/tui"
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
	var under, openFlag string
	var dryRun, agent, asJSON bool
	cmd := &cobra.Command{
		Use:   "new <kind> <title...>",
		Short: "Make the next card: a new root, or the next child of --under KEY",
		Long: `Make the next card. With --under KEY it is KEY's next child (6 gives 6a, 6b;
6a gives 6a1, 6a2); without, the next root.

The kinds come from the box's spec; zk config lists them. On a terminal, with
the kind or the title left out, zk new asks for them: a kind, a title, then a
parent. With every argument given, it never prompts.

--open says where the new card opens: obsidian (the default when zk new asked
for the card), editor ($VISUAL, else $EDITOR), or none (the default when every
argument was given). If it cannot be opened, the card stays made; on a terminal
zk offers the editor instead.`,
		Example: `  zk new
  zk new idea Taste is deletion
  zk new claim "Judges decide" --under 5 --open editor
  zk new idea Another thought --under 6a --dry-run`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			where := open.None
			if openFlag != "" {
				if where, err = open.Parse(openFlag); err != nil {
					return UsageError{err}
				}
			}
			var c box.NewCard
			if len(args) < 2 {
				if !app.Interactive() {
					what := "a kind and a title"
					if len(args) == 1 {
						what = "a title"
					}
					fmt.Fprintf(app.Err, "kinds in this box:\n%s", kindList(s))
					return usagef("zk new needs %s", what)
				}
				res, err := app.NewUI(uiInput(app, s, args, under))
				if err != nil {
					return err
				}
				if res.Cancelled {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "zk new: cancelled, nothing made")
					return err
				}
				c = box.NewCard{Kind: res.Kind, Title: res.Title, Under: res.Under}
				if openFlag == "" {
					where = open.Obsidian
				}
			} else {
				title, err := box.CleanTitle(args[1:])
				if err != nil {
					return err
				}
				c = box.NewCard{Kind: args[0], Title: title, Under: under}
				if _, ok := s.Spec.Kinds[c.Kind]; !ok {
					fmt.Fprintf(app.Err, "kinds in this box:\n%s", kindList(s))
				}
			}
			c.Agent, c.Now = agent, app.Now()

			var k, p string
			if dryRun {
				k, p, err = s.PlanCard(c)
			} else {
				k, p, err = s.MakeCard(c)
			}
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if asJSON {
				err = jsonLine(out, map[string]any{"key": k, "path": s.Rel(p), "dry_run": dryRun})
			} else if dryRun {
				_, err = fmt.Fprintln(out, "would make "+s.Rel(p))
			} else {
				_, err = fmt.Fprintln(out, s.Rel(p))
			}
			if err != nil || dryRun {
				return err
			}
			return openCard(app, where, p)
		},
	}
	cmd.Flags().StringVar(&under, "under", "", "make it the next child of this card")
	cmd.Flags().StringVar(&openFlag, "open", "", "where to open the new card: obsidian, editor or none")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the card it would make, and write nothing")
	cmd.Flags().BoolVar(&agent, "agent", false, "an agent made this card (sets authored_by)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print one JSON object")
	return cmd
}

// uiInput is what the zk new UI needs from the box and the command line.
func uiInput(app *App, s *box.Store, args []string, under string) tui.Input {
	in := tui.Input{Box: s.Spec.Name, Reason: app.reason(), Under: under, AskParent: under == ""}
	for _, name := range s.Spec.KindNames() {
		in.Kinds = append(in.Kinds, tui.Kind{Name: name, Description: s.Spec.Kinds[name].Description})
	}
	if rows, err := s.List(box.Filter{}); err == nil {
		for _, r := range rows {
			in.Cards = append(in.Cards, tui.Card{Key: r.Key, Title: r.Title, Kind: r.Kind, Status: r.Status, Level: r.Level})
		}
	}
	if len(args) > 0 {
		if _, ok := s.Spec.Kinds[args[0]]; ok {
			in.Kind = args[0]
		}
	}
	return in
}

// openCard opens a card that has been made. If that fails, the card stays,
// and on a terminal zk offers the editor instead (ADR 261115-mjrm).
func openCard(app *App, where open.Where, path string) error {
	err := app.Opener.Open(where, path)
	if err == nil {
		return nil
	}
	editor := app.Opener.EditorName()
	if where == open.Editor || !app.Interactive() || editor == "" {
		return fmt.Errorf("the card is made, but %w", err)
	}
	fmt.Fprintf(app.Err, "The card is made, but %v.\nOpen it in %s instead? [y/N] ", err, editor)
	answer, _ := bufio.NewReader(app.In).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		return nil
	}
	if err := app.Opener.Open(open.Editor, path); err != nil {
		return fmt.Errorf("the card is made, but %w", err)
	}
	return nil
}

func newListCmd(app *App) *cobra.Command {
	var f box.Filter
	var asTree, ascii, asJSON bool
	var openFlag string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List cards depth first",
		Long: `List cards depth first, or with --tree as a tree.

On a terminal, zk list opens the interactive list: the cards on a full screen
that you scroll, filter by typing after /, switch between rows and the tree
with t, and fold by branch in the tree (left and right for one branch, with
shift for the whole family, - and + for all of them). Enter opens the card under the cursor in Obsidian, or in the
editor with --open editor, and comes back to the list; e opens it in the
editor; q, esc or ctrl+c quits.

With no terminal, --json, --no-input or ZK_NO_INPUT=1, zk list prints the
cards and exits.`,
		Args: noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := app.Open()
			if err != nil {
				return err
			}
			if f.Status != "" && !contains(box.CardStatuses, f.Status) {
				return usagef("--status must be one of %s", strings.Join(box.CardStatuses, ", "))
			}
			where := open.Obsidian
			if openFlag != "" {
				if where, err = open.Parse(openFlag); err != nil || where == open.None {
					return usagef("--open must be obsidian or editor, not %q", openFlag)
				}
			}
			rows, err := s.List(f)
			if err != nil {
				return err
			}
			// The tree's root line is the box, or the --under card itself.
			root := s.Spec.Name
			if f.Under != "" && (asTree || app.Interactive()) {
				if all, err := s.List(box.Filter{}); err == nil {
					for _, r := range all {
						if r.Key == f.Under {
							root = cardLine(r)
						}
					}
				}
			}
			glyphs := tree.Unicode
			if ascii {
				glyphs = tree.ASCII
			}
			if !asJSON && app.Interactive() && app.ListUI != nil {
				return app.ListUI(listInput(app, s, rows, root, asTree, glyphs, where))
			}
			out := cmd.OutOrStdout()
			if asTree && !asJSON {
				_, err := fmt.Fprint(out, treeText(drawTree(root, rows, glyphs)))
				return err
			}
			for _, r := range rows {
				if asJSON {
					if err := jsonLine(out, r); err != nil {
						return err
					}
					continue
				}
				fmt.Fprintln(out, rowLine(r))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&f.Status, "status", "", "only cards with this status (raw, proposed, linked)")
	cmd.Flags().StringVar(&f.Kind, "kind", "", "only cards of this kind")
	cmd.Flags().StringVar(&f.Under, "under", "", "only the descendants of this card (with --tree, drawn under it)")
	cmd.Flags().BoolVar(&asTree, "tree", false, "draw the cards as a tree, like the tree command")
	cmd.Flags().BoolVar(&ascii, "ascii", false, "draw the tree's lines with plain characters")
	cmd.Flags().BoolVar(&asJSON, "json", false, "one JSON object per card")
	cmd.Flags().StringVar(&openFlag, "open", "", "where enter opens a card in the interactive list: obsidian (the default) or editor")
	return cmd
}

// rowLine is how one card reads in the rows of zk list.
func rowLine(r box.Row) string {
	return fmt.Sprintf("%-10s %-9s %-9s %s", r.Key, r.Status, r.Kind, r.Title)
}

// listInput is what the interactive list needs from the box and the command
// line (ADR 261115-9pqw).
func listInput(app *App, s *box.Store, rows []box.Row, root string, asTree bool, glyphs tree.Glyphs, where open.Where) tui.ListInput {
	in := tui.ListInput{Box: s.Spec.Name, Reason: app.reason(), Root: root, Tree: asTree, Glyphs: glyphs,
		Where: "Obsidian", Attached: where == open.Editor, Editor: app.Opener.EditorName()}
	if where == open.Editor {
		in.Where = in.Editor
	}
	paths := make(map[string]string, len(rows))
	for _, r := range rows {
		paths[r.Key] = s.CardPath(r.Key)
		in.Cards = append(in.Cards, tui.ListCard{
			Card:   tui.Card{Key: r.Key, Title: r.Title, Kind: r.Kind, Status: r.Status, Level: r.Level},
			Row:    rowLine(r),
			Branch: cardLine(r),
		})
	}
	in.Open = func(c tui.ListCard) error { return app.Opener.Open(where, paths[c.Key]) }
	in.Edit = func(c tui.ListCard) error { return app.Opener.Open(open.Editor, paths[c.Key]) }
	return in
}

// reason is why the box was chosen, with the home directory written ~.
func (app *App) reason() string {
	reason := app.resolution.Reason
	if env, err := app.Env(); err == nil && env.Home != "" {
		reason = strings.ReplaceAll(reason, env.Home+string(filepath.Separator), "~"+string(filepath.Separator))
	}
	return reason
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
	var fix, fixTypes, strict, asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check cards and sources against the box's spec; exits 1 on an error",
		Long: `Check cards and sources against the box's spec, and the types Obsidian gives
the box's fields (.obsidian/types.json). Exits 1 on an error; warnings (a
duplicate nickname, a skipped key, a field with no type) do not fail unless
--strict. A field Obsidian holds as another type is an error.

With --fix, first write the derived fields (key, level, sort, the key's tag,
and aliases[0] from the H1), then check.

With --fix-types, first add the missing types to .obsidian/types.json, then
check. It never changes a type that is set. Reload Obsidian afterwards, or it
may write its own copy over the file.`,
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
			if fixTypes {
				want := s.Spec.Mapping.PropertyTypes()
				added, _, err := box.AddTypes(s.Spec.Vault, want)
				if err != nil {
					return err
				}
				for _, name := range added {
					if asJSON {
						jsonLine(out, map[string]any{"added_type": name, "type": want[name], "path": box.TypesFile})
					} else {
						fmt.Fprintf(out, "added %s: %s to %s\n", name, want[name], box.TypesFile)
					}
				}
				if len(added) > 0 && !asJSON {
					fmt.Fprintln(out, "reload Obsidian so it reads the new types; while it runs it may write its own copy over them")
				}
			}
			return report(app, s, s.Check(strict), asJSON)
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "write the derived fields first")
	cmd.Flags().BoolVar(&fixTypes, "fix-types", false, "add the missing property types to .obsidian/types.json first")
	cmd.Flags().BoolVar(&strict, "strict", false, "a duplicate nickname or a skipped key is an error, not a warning")
	cmd.Flags().BoolVar(&asJSON, "json", false, "one JSON object per finding")
	return cmd
}

func report(app *App, s *box.Store, findings []box.Finding, asJSON bool) error {
	out := app.Out
	errs := 0
	var fixable []string
	seen := map[string]bool{}
	missingTypes := 0
	for _, f := range findings {
		if !f.Warning {
			errs++
		}
		if f.Rule == "property-type" && f.Fixable {
			missingTypes++
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
		if missingTypes > 0 {
			fmt.Fprintf(out, "%d field(s) with no property type: run `zk check --fix-types`\n", missingTypes)
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
