---
id: "261115-v1yz"
title: Say on stderr which box was chosen when it was not found from here
summary: When ZK_BOX or the default chose the box, zk ends with a stderr line naming it, its folder and why. Read before changing output or box resolution.
status: proposed
date: 2026-11-15
scope: [box resolution, output]
authored_by: agent
---

# Say on stderr which box was chosen when it was not found from here

## Context

ADR 261115-zd6s resolves a box in a fixed order: `--box`, `ZK_BOX`, the nearest `zk.toml` walking up from the working directory, then the user config's `default`. Only `zk config` says which step chose the box. Run from a folder outside any box, `zk list --tree` silently acts on the default box: the maintainer ran it from another repository and got his vault's box with no sign of where it came from. The tree's first line is the box's `name`, not where it is, and with many cards it scrolls away.

## Considered Options

1. **A line on stderr before the output.** Seen first, but a long list or tree scrolls it off the screen.
2. **A line on stderr after the output.** It sits just above the prompt, where the eye lands when a command ends, however long the output.
3. **A line on every run.** Always the same, but noise when the box is obviously the one you are standing in, and soon ignored.
4. **Change the tree's header to the box's path.** Covers only `list --tree`, and changes stdout, which scripts read.

## Decision

Option 2, only when the box was not found from here: when `ZK_BOX` or the user config's `default` chose it. A box found walking up from the working directory, or named with `--box`, gets no line. The line is:

```text
zk: box <name> at <folder> (<why>)
```

with the home directory written `~`, for example `zk: box pkm at ~/vault/Slip Box (the default in ~/.config/zk/config.toml)`. It is written after an error too, so an error is still the first thing on stderr. It is written only when stderr is a terminal, and never with `--json`, so scripts and agents see no change. `zk config` already says all of this, and does not resolve through the same path, so it prints no line.

## Consequences

- Running zk from outside a box says which box it touched, on every command that opens one.
- Stdout is unchanged; a pipe or a file sees exactly what it saw before.
- A command run with stderr redirected loses the line. That is deliberate: a script knows which box it asked for.
- If the vault-root search of work item 261115-7m9b lands, a box it finds is "not found from here" in the same sense, and should get the same line.

## Relations

- Related to: 261115-zd6s (box resolution), 261115-czb2 (terminal-only behaviour)
