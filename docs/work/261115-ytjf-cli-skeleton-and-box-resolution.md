---
id: "261115-ytjf"
title: Build the CLI skeleton, user config and box resolution
status: active
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
kind: feature
summary: Cobra app with the command tree, TOML user config at the XDG path, box resolution order, and zk config to explain the choice.
claimed_by: claude-code-opus-5-5
claimed_at: 2026-11-15T23:20
---

# Build the CLI skeleton, user config and box resolution

## Goal

Done when:

- [ ] `zk --help`, `zk version` and `zk completion` work.
- [ ] The user config is read from `$ZK_CONFIG`, `$XDG_CONFIG_HOME/zk/config.toml` or `~/.config/zk/config.toml`, never merged.
- [ ] A box is chosen by `--box`, `ZK_BOX`, the nearest spec walking up from the working directory (stopping at `$HOME`), then the default; `zk config` prints the box and the reason.
- [ ] Tests cover each step of the resolution order.

## Next

Start the Go module (github.com/alexsasharegan/zk) with Cobra: root command, version and completion, then the user config loader and the box resolution order from ADR 261115-zd6s, with tests for each step.

## Log

- 2026-11-15: filed.
- 2026-11-15 16:20-07:00: claimed (claude-code-opus-5-5).
