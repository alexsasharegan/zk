---
id: "261115-93kx"
title: Make zk list interactive on a terminal
status: blocked
kind: feature
created: 2026-11-15
updated: 2026-11-15
summary: On a terminal, zk list opens an interactive list that filters, folds the tree and opens a card; piped or with --json or --no-input it prints as before.
blocked_by: [261115-9pqw]
---

# Make zk list interactive on a terminal

## Goal

`zk list` on a box of a few hundred cards scrolls its top off the screen. On a terminal it should open an interactive list of the cards instead: a full-screen view you scroll and quit, built on the terminal UI `zk new` already has (ADR 261115-9pqw).

Done when:

- [ ] On a terminal, `zk list` opens the interactive list, with `--status`, `--kind` and `--under` applied and `--tree` starting it in the tree view; with no terminal, `--json`, `--no-input` or `ZK_NO_INPUT=1` its output is unchanged.
- [ ] Typing filters the cards by key, title, kind and status, with the search `zk new` uses for a parent, shared rather than copied.
- [ ] A key switches between rows and the tree, and in the tree a branch folds and unfolds.
- [ ] Enter opens the card in Obsidian, or in the editor with `--open editor` (ADR 261115-mjrm), and returns to the list; q, esc or ctrl+c quits and leaves the screen clean.
- [ ] The header names the box and why it was chosen, and the stderr line of ADR 261115-v1yz is printed after the list closes.
- [ ] Tests cover the interactive list and every plain path, the README describes it, and the maintainer has tried it on his own box.

## Next

Blocked until the maintainer accepts or changes ADR 261115-9pqw; then set it ready. Then read internal/tui/newcard.go and pull its parent search into a shared piece, and design the keys (switching view, folding) before building. Comes after zk init (261115-h674).

## Log

- 2026-11-15: filed. The maintainer asked for a pager for `zk list` built on the terminal UI; an interactive list was chosen over `$PAGER` and a Bubble Tea text pager (ADR 261115-9pqw, proposed).
- 2026-11-15: renamed from "browser" to "interactive list", which the maintainer found clearer.
