---
id: "261115-w9sn"
title: Build zk to parity with slip
status: ready
kind: feature
created: 2026-11-15
updated: 2026-11-15
summary: "Replace the slip proof of concept with zk: design, CLI, box spec, commands, demo vault, then the TUI."
---

# Build zk to parity with slip

## Goal

Replace the Python `slip` proof of concept with `zk`, so Alexander's slip box can move into his main Obsidian vault. Children, in order:

- [Draft the zk design ADRs](done/261115-q0r0-design-adrs.md): the contract for the rest.
- [Build the CLI skeleton, user config and box resolution](261115-ytjf-cli-skeleton-and-box-resolution.md)
- [Read the box spec and apply the field mapping](261115-ahma-box-spec-and-field-mapping.md)
- [Port the slip commands to zk](261115-wfyq-card-commands-at-parity.md)
- [Build the demo vault and the parity suite](261115-823p-demo-vault-and-parity-suite.md)
- [Build the zk new terminal UI](261115-ct75-tui-new.md)

Migrating his own box is tracked outside this repository, after this.

Done when:

- [ ] Every child is done or dropped.
- [ ] `zk check` passes on a copy of a real, migrated box.

## Next

Work the next child, 261115-ytjf (CLI skeleton, user config and box resolution).

## Log

- 2026-11-15: filed.
