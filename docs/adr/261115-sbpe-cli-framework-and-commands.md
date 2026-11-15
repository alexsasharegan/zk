---
id: "261115-sbpe"
title: zk is a Cobra CLI with a fixed command set
summary: zk is one Go binary built on Cobra and Bubble Tea, with no Viper, and these command names. Read before adding a command or a dependency.
status: accepted
date: 2026-11-15
scope: [cmd, the command set, dependencies]
authored_by: agent
---

# zk is a Cobra CLI with a fixed command set

## Context

`zk` replaces `slip`, a Python script of about 750 lines in a private repository that Alexander treats as a proof of concept. He wants a hardened Go tool with the properties of a Cobra CLI (subcommands, generated help and shell completion, consistent flags) and a terminal UI for the steps where he writes by hand. The slip box it serves is moving into his main Obsidian vault. Its configuration is small: where boxes are, and each box's own spec (ADRs 261115-zd6s and 261115-0tp4).

## Considered Options

- **Cobra with Viper.** Viper reads config files, environment and flags into one merged view. That merging is the behaviour ADR 261115-zd6s rules out, and Viper is heavy for a handful of paths.
- **The standard library `flag` package.** No dependencies, but subcommands, help and completion would be hand-written.
- **Cobra, a TOML reader, and Bubble Tea for the UI.** Chosen.
- **Doing nothing**, keeping `slip`. It works, but Alexander does not want it as the final tool, and it cannot be configured for another vault.

## Decision

- One Go module, `github.com/alexsasharegan/zk`, one binary, `zk`.
- Dependencies: `github.com/spf13/cobra`; a TOML reader (`github.com/BurntSushi/toml`); `github.com/charmbracelet/bubbletea` with `bubbles` and `lipgloss` for the UI; `golang.org/x/term` for terminal detection. Anything else needs Alexander's word.
- Commands, at parity with `slip` and renamed where a convention is clearer (agreed 2026-11-15):

| Command | Does | Was |
|---|---|---|
| `zk new <kind> <title> [--under KEY]` | Make the next card, a new root or the next child of KEY | `slip new` |
| `zk list` (alias `ls`) | List cards, filtered by `--status`, `--kind`, `--under`, `--tree` | `slip ls` |
| `zk show KEY` | A card's key facts (level, sort key, tag, file, next child) and its links in and out | `slip key`, `slip links` |
| `zk retitle KEY <title>` | Change the H1 and nickname together | `slip retitle` |
| `zk check [--fix] [--strict]` (alias `fix` for `check --fix`) | Validate the box; `--fix` writes the derived fields | `slip check`, `slip fix` |
| `zk source new <slug>` | Make a source page | `slip source new` |
| `zk source done <slug> KEY...` | Mark a source extracted, with its cards | `slip source done` |
| `zk config` | Print the chosen box and why | new |
| `zk version`, `zk completion` | Cobra's own | new |

- `--json` prints one JSON object per line (JSON Lines) on commands that list.
- Exit codes: 0 success, 1 findings (`check`) or a failed operation, 2 usage error.

## Consequences

- Help and completion come from Cobra, so command docs stay in the code.
- The command set is a contract with agents and Alexander's habits; renaming a command after release needs a new ADR, and old names become aliases.
- Trails are not in the command set yet (none exist yet, and their fields are not settled).

## Confirmation

- `zk --help` lists exactly the commands above.
- `go list -m all` shows no dependency outside the list above and their own dependencies.

## Review

Revisit when trails get commands, or when a second front end (an Obsidian plugin) needs the same operations.

## Revert

Before release, edit freely. After release, supersede with a new ADR and keep the old names as aliases.
