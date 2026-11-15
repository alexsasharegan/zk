---
id: "261115-wfyq"
title: Port the slip commands to zk
status: active
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
kind: feature
summary: Keys, sort keys and the commands new, list, show, retitle, check (with --fix) and source new and done, at parity with slip.
claimed_by: claude-code-opus-5-5
claimed_at: 2026-11-15T23:36
---

# Port the slip commands to zk

## Goal

Done when:

- [ ] Key parsing, next sibling and next child, level, sort key and tag match `slip`.
- [ ] `new`, `list` (alias `ls`), `show`, `retitle`, `check` (`--fix`, alias `fix`), `source new` and `source done` work with full arguments and never prompt.
- [ ] `--json` prints JSON Lines on the listing commands.
- [ ] Without a terminal, a missing argument exits non-zero with usage; `--no-input` and `ZK_NO_INPUT` force that on a terminal.

## Next

Record the go.yaml.in/yaml/v3 dependency in an ADR, then write internal/frontmatter (read and write a note's front matter, keeping key order, quoting and the body), then keys, then the commands.

## Log

- 2026-11-15: filed.
- 2026-11-15 16:36-07:00: claimed (claude-code-opus-5-5).
