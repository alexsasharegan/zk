---
id: "261115-93kx"
title: Make zk list interactive on a terminal
status: active
kind: feature
created: 2026-11-15
updated: 2026-11-15
summary: On a terminal, zk list opens an interactive list that filters, folds the tree and opens a card; piped or with --json or --no-input it prints as before.
claimed_by: claude-code-opus-5-5
claimed_at: 2026-11-16T06:59
---

# Make zk list interactive on a terminal

## Goal

`zk list` on a box of a few hundred cards scrolls its top off the screen. On a terminal it should open an interactive list of the cards instead: a full-screen view you scroll and quit, built on the terminal UI `zk new` already has (ADR 261115-9pqw, accepted).

Done when:

- [x] On a terminal, `zk list` opens the interactive list, with `--status`, `--kind` and `--under` applied and `--tree` starting it in the tree view; with no terminal, `--json`, `--no-input` or `ZK_NO_INPUT=1` its output is unchanged.
- [x] Typing filters the cards by key, title, kind and status, with the search `zk new` uses for a parent, shared rather than copied.
- [x] A key switches between rows and the tree, and in the tree a branch folds and unfolds.
- [x] Enter opens the card in Obsidian, or in the editor with `--open editor` (ADR 261115-mjrm), and returns to the list; q, esc or ctrl+c quits and leaves the screen clean.
- [x] The header names the box and why it was chosen, and the stderr line of ADR 261115-v1yz is printed after the list closes.
- [x] Tests cover the interactive list and every plain path, the README describes it, the glossary's interactive list entry is no longer marked provisional (and still matches what was built), and the maintainer has tried it on his own box.

## Next

The maintainer tried it on his own box ("working beautifully") and asked for colour and fold all, which are built. He looks at those two, then the pull request can be merged and this item closed. Offered, not built: number keys that show the tree to a level.

## Log

- 2026-11-15: filed. The maintainer asked for a pager for `zk list` built on the terminal UI; an interactive list was chosen over `$PAGER` and a Bubble Tea text pager (ADR 261115-9pqw, proposed).
- 2026-11-15: renamed from "browser" to "interactive list", which the maintainer found clearer.
- 2026-11-15: the glossary now defines terminal UI, interactive list (provisional) and plain output, so the names come before the code, as the maintainer asked.
- 2026-11-15: the maintainer accepted ADR 261115-9pqw as written; ready.
- 2026-11-15 23:59-07:00: claimed (claude-code-opus-5-5).
- 2026-11-16: the maintainer approved the keys as proposed: `/` filters (not type-to-filter, since q quits), `t` switches rows and tree, left/right/space fold, enter opens, `e` opens in the editor, q/esc/ctrl+c quits. Rows, tree and fold are in the glossary.
- 2026-11-16: built. internal/tree lays the tree out for both `zk list --tree` and the list; internal/tui/search.go is the one card search (zk new's parent step now matches kind and status too); internal/tui/list.go is the model; `App.ListUI` injects it. `zk list` gained `--open obsidian|editor`. A failed open shows a notice that offers `e` in place of the y/N question of ADR 261115-mjrm, since the list has the screen. Driven in tmux on the demo box: fold, filter, the editor round trip and a clean quit all worked; opening in Obsidian was not driven. The list does not reload after the editor closes.
- 2026-11-16: the maintainer tried it on his own box: it works, and he asked for three things. Built: the key is coloured and the kind, status and tree lines are faint (the cursor's line stays plain, reversed); `-` folds every branch and `+` or `=` unfolds them all, the recursive version he described; the fold count `[+3]` is bold and coloured, since it is the only sign there is more to unfold.
