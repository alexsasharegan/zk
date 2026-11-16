---
id: "261115-9pqw"
title: zk list is interactive on a terminal
summary: On a terminal, zk list opens an interactive list that filters, folds the tree and opens cards; piped or with --json or --no-input it prints as before.
status: proposed
date: 2026-11-15
scope: [zk list, terminal UI, internal/tui]
authored_by: agent
---

# zk list is interactive on a terminal

## Context

`zk list` prints every card, and `zk list --tree` draws them as a tree. With a box of a few hundred cards the output is longer than the window, and the top scrolls away. The maintainer asked for a pager, and suggested building it on the terminal UI that `zk new` already has (internal/tui, Bubble Tea) rather than calling out to `less`.

ADR 261115-czb2 opens a UI only when stdin and stdout are terminals and a required argument is missing. `zk list` has no required argument, so it can never meet that rule. czb2 asks to be revisited "when a second command gets a UI"; this is that review, and czb2 itself stays as it is.

Two other records touch this. ADR 261115-v1yz prints a stderr line naming the box after a command's output when the box was not found from the working directory. ADR 261115-mjrm decides how a card is opened: Obsidian by default after the UI, or the editor with `--open editor`.

## Considered Options

1. **Pipe through `$PAGER`, as git does.** When stdout is a terminal and the output is taller than the window, run `$PAGER` (`less -FRX` by default), with `--no-pager` to skip it. Small, and it uses the pager the user knows, but it only scrolls and searches text.
2. **A Bubble Tea viewport that pages the text.** It only scrolls and searches, like `less` but with fewer keys and habits to learn.
3. **An interactive list: `zk list` made interactive.** A full-screen list of cards in the terminal, as rows or as the tree, that can be filtered by typing, folded by branch, and opened with enter.
4. **The interactive list as its own command, such as `zk browse`.** No change to `zk list`, but two names for one view of the box, the reason czb2 rejected `zk ui new`.
5. **Doing nothing.** `zk list | less` works today.

## Decision

Option 3.

`zk list` opens the interactive list when stdin and stdout are both terminals, and none of `--json`, `--no-input` or `ZK_NO_INPUT=1` is given. Otherwise it prints exactly what it prints today, so agents, scripts and pipes see no change. This adds a second rule to czb2's: a command whose whole job is to show the box may open its UI on a terminal even when nothing is missing.

In the interactive list:

- the rows are what `zk list` would print, with the same `--status`, `--kind` and `--under` filters applied first; `--tree` starts it in the tree view, and a key switches between the two;
- typing filters by key, title, kind and status, with the search `zk new` uses to choose a parent;
- in the tree view a branch folds and unfolds;
- enter opens the card as ADR 261115-mjrm says, Obsidian by default and `--open editor` for the editor, then returns to the list;
- q, esc or ctrl+c quits, leaving nothing on the screen.

When the box was not found from the working directory, the list's header names it and why, as `zk new`'s header does, and the stderr line of ADR 261115-v1yz is printed after the list closes.

The other long outputs (`zk show`, `zk check`) stay plain text. A `$PAGER` for them is a separate decision, if one is wanted.

## Consequences

- Run by hand on a terminal, `zk list` no longer prints and returns. To get the text there, pass `--no-input` or pipe it (`zk list | cat`).
- Agents see no change: their shells have no terminal, and one that runs on a terminal can pass `--no-input`, as ADR 261115-czb2 asks the agent skill for the box to say.
- Each mode needs a test: the interactive list on a terminal, and the plain output with no terminal, `--json` and `--no-input`.
- internal/tui gets a second model; the parent search from `zk new` should become a shared piece rather than a copy.
- No new dependency: `bubbles` already has the list and viewport pieces.
- The glossary defines terminal UI, interactive list and plain output, so the names are settled before the code. The interactive list entry is marked provisional until it is built.

## Relations

- Related to: 261115-czb2 (when zk prompts), 261115-mjrm (opening a card), 261115-v1yz (the box notice on stderr)
