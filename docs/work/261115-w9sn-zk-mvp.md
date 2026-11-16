---
id: "261115-w9sn"
title: Build zk to parity with slip
status: blocked
kind: feature
created: 2026-11-15
updated: 2026-11-15
summary: "Replace the slip proof of concept with zk: design, CLI, box spec, commands, demo vault, then the TUI."
blocked_by: [the move of the maintainer's box into his vault (outside this repository)]
---

# Build zk to parity with slip

## Goal

Replace the Python `slip` proof of concept with `zk`, so Alexander's slip box can move into his main Obsidian vault. Children, in order:

- [Draft the zk design ADRs](done/261115-q0r0-design-adrs.md): the contract for the rest.
- [Build the CLI skeleton, user config and box resolution](done/261115-ytjf-cli-skeleton-and-box-resolution.md)
- [Read the box spec and apply the field mapping](done/261115-ahma-box-spec-and-field-mapping.md)
- [Port the slip commands to zk](done/261115-wfyq-card-commands-at-parity.md)
- [Build the demo vault and the parity suite](done/261115-823p-demo-vault-and-parity-suite.md)
- [Build the zk new terminal UI](done/261115-ct75-tui-new.md)

Migrating his own box is tracked outside this repository, after this.

Done when:

- [x] Every child is done or dropped.
- [ ] `zk check` passes on a copy of a real, migrated box.

## Next

Blocked until Alexander's box has moved into his vault (tracked outside this repository). Then run zk check on a copy of it, tick the last box and close this item.

## Log

- 2026-11-15: filed.
- 2026-11-15 17:48-07:00: ticked "Every child is done or dropped.": every child closed (7/7); see docs/work/done
- 2026-11-15 17:48-07:00: MVP built and merged (PRs 1 to 7): design ADRs, CLI and box resolution, box spec and field mapping, YAML front matter, the slip commands at parity, CI, the demo vault with the parity suite, and the zk new UI. Alexander tested it on the demo vault (Obsidian open, editor open, Obsidian launched from closed); list --tree followed (261115-x79t, PR 9). The install step after a merge is in AGENTS.md (PR 8). The last box waits on the move of his real box into his vault, tracked outside this repository.
