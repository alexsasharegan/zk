---
id: "261115-ytjf"
title: Build the CLI skeleton, user config and box resolution
status: ready
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
kind: feature
summary: Cobra app with the command tree, TOML user config at the XDG path, box resolution order, and zk config to explain the choice.
---

# Build the CLI skeleton, user config and box resolution

## Goal

Done when:

- [ ] `zk --help`, `zk version` and `zk completion` work.
- [ ] The user config is read from `$ZK_CONFIG`, `$XDG_CONFIG_HOME/zk/config.toml` or `~/.config/zk/config.toml`, never merged.
- [ ] A box is chosen by `--box`, `ZK_BOX`, the nearest spec walking up from the working directory (stopping at `$HOME`), then the default; `zk config` prints the box and the reason.
- [ ] Tests cover each step of the resolution order.

## Log

- 2026-11-15: filed.
