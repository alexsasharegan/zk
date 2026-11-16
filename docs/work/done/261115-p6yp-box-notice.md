---
id: "261115-p6yp"
title: Say which box was chosen when it was not found from here
status: done
kind: feature
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
---

# Say which box was chosen when it was not found from here

## Goal

Run outside a box, zk silently acts on the user config's default box (or `ZK_BOX`), and only `zk config` says so. The maintainer ran `zk list --tree` from another repository and could not tell which box it read. ADR 261115-v1yz (proposed): end the output with one stderr line naming the box, its folder and why, when the box was not found from the working directory; only on a terminal, never with `--json`.

Done when:

- [x] `zk list` and the other box commands print the line last on stderr when `ZK_BOX` or the default chose the box.
- [x] No line for a box found walking up, for `--box`, when stderr is not a terminal, with `--json`, or for `zk config`.
- [x] Tests cover each case, including a failing command (the line comes after the error).
- [x] The README's "Choosing a box" mentions the line.

## Next

## Log

- 2026-11-15: filed.
- 2026-11-15: built: `App.noteBox` in `internal/cli/root.go`, called from `Run` after any error; `App.ErrTerminal`; tests in `internal/cli/notice_test.go`.
- 2026-11-15 20:18-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15: merged as PR 12; ADR 261115-v1yz is accepted with the merge of the PR that closes this item.

## Outcome

Run outside a box, zk now ends with a stderr line naming the box it used, when `ZK_BOX` or the default chose it.
