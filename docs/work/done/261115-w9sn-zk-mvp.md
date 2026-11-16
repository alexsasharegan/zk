---
id: "261115-w9sn"
title: Build zk to parity with slip
status: done
kind: feature
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
summary: "Replace the slip proof of concept with zk: design, CLI, box spec, commands, demo vault, then the TUI."
blocked_by: [the move of the maintainer's box into his vault (outside this repository)]
---

# Build zk to parity with slip

## Goal

Replace the Python `slip` proof of concept with `zk`, so Alexander's slip box can move into his main Obsidian vault. Children, in order:

- [Draft the zk design ADRs](261115-q0r0-design-adrs.md): the contract for the rest.
- [Build the CLI skeleton, user config and box resolution](261115-ytjf-cli-skeleton-and-box-resolution.md)
- [Read the box spec and apply the field mapping](261115-ahma-box-spec-and-field-mapping.md)
- [Port the slip commands to zk](261115-wfyq-card-commands-at-parity.md)
- [Build the demo vault and the parity suite](261115-823p-demo-vault-and-parity-suite.md)
- [Build the zk new terminal UI](261115-ct75-tui-new.md)

Migrating his own box is tracked outside this repository, after this.

Done when:

- [x] Every child is done or dropped.
- [x] `zk check` passes on a copy of a real, migrated box.

## Next

## Log

- 2026-11-15: filed.
- 2026-11-15 17:48-07:00: ticked "Every child is done or dropped.": every child closed (7/7); see docs/work/done
- 2026-11-15 17:48-07:00: MVP built and merged (PRs 1 to 7): design ADRs, CLI and box resolution, box spec and field mapping, YAML front matter, the slip commands at parity, CI, the demo vault with the parity suite, and the zk new UI. Alexander tested it on the demo vault (Obsidian open, editor open, Obsidian launched from closed); list --tree followed (261115-x79t, PR 9). The install step after a merge is in AGENTS.md (PR 8). The last box waits on the move of his real box into his vault, tracked outside this repository.
- 2026-11-15: Alexander's box (157 cards, 15 sources) moved into his vault. `zk check --strict`, built from this branch, finds nothing on a copy of it (with the one vault note a source links to).

## Outcome

zk replaces `slip`: the MVP is built, and a real migrated box passes `zk check --strict`.
