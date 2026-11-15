---
id: "261115-zd6s"
title: One user config locates boxes; a box is resolved in a fixed order
summary: Where the user config lives, what it holds, and the order zk uses to choose a box. Read before touching config loading.
status: accepted
date: 2026-11-15
scope: [config, box resolution]
authored_by: agent
---

# One user config locates boxes; a box is resolved in a fixed order

## Context

A slip box is a folder in an Obsidian vault. The author has one today in his main vault, the repository will ship a demo vault with another, so `zk` has to know which box a command acts on. He wants the config in the usual dotfile place, and he asked whether running `zk` inside a box should pick that box. Layered config resolution is a common source of bugs, mostly because settings from several files are merged.

## Considered Options

- **Only a user config with a default box.** Simple, but working in the demo vault, or any second box, means passing `--box` every time.
- **Settings merged from a local file and the user file** (the git config model). Flexible, and the source of the bugs above: a value's origin becomes hard to see.
- **A user config that only names boxes, plus discovery of the nearest box, with no merging.** Chosen.

## Decision

- **Location.** The user config is the first that exists of: `$ZK_CONFIG`; `$XDG_CONFIG_HOME/zk/config.toml`; `~/.config/zk/config.toml`. Go's `os.UserConfigDir` is not used, because on macOS it is `~/Library/Application Support`.
- **Contents.** Named boxes, each the path of its spec file (ADR 261115-0tp4), and which one is the default. Nothing else for now:

```toml
default = "notes"

[boxes.notes]
spec = "~/notes/Slip Box/zk.toml"

[boxes.demo]
spec = "~/github.com/alexsasharegan/zk/examples/demo-vault/Slip Box/zk.toml"
```

- **Resolution order**, first match wins:
  1. `--box NAME` (a name from the user config) or `--box PATH` (a spec file or its folder).
  2. `ZK_BOX`, read the same way.
  3. The nearest `zk.toml` found by walking up from the working directory, stopping at `$HOME` or the filesystem root.
  4. `default` in the user config.
  5. Otherwise an error that says how to name a box.
- **No merging.** The user config locates boxes; everything about a box is in its spec. No value is read from two files.
- `zk config` prints the chosen box, its spec path and which step chose it (for example "found walking up from the working directory"). The terminal UI shows the same in its header.

## Consequences

- Inside a vault, `zk` works with no flags; elsewhere it uses the default.
- A box missing from the user config still works by path or discovery, so the demo vault needs no setup.
- A spec file that does not sync to another device makes discovery fail there; the error says so (see ADR 261115-0tp4).

## Confirmation

- Tests cover each resolution step, including a walk that stops at `$HOME`, and that `--box` beats a discovered box.
- `zk config` names the step in every case.

## Review

Revisit if a setting that is truly per user (not per box) turns up, such as an editor override.

## Revert

Dropping discovery leaves steps 1, 2 and 4; no stored data changes.
