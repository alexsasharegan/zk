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
decisions: [261115-t9jd]
---

# Port the slip commands to zk

## Goal

Done when:

- [ ] Key parsing, next sibling and next child, level, sort key and tag match `slip`.
- [ ] `new`, `list` (alias `ls`), `show`, `retitle`, `check` (`--fix`, alias `fix`), `source new` and `source done` work with full arguments and never prompt.
- [ ] `--json` prints JSON Lines on the listing commands.
- [ ] Without a terminal, a missing argument exits non-zero with usage; `--no-input` and `ZK_NO_INPUT` force that on a terminal.

## Next

Write internal/key (parse, next sibling, next child, level, sort key, tag) to match slip, then the commands. This branch is cut from box-spec (PR 4): push it only after PR 4 is merged, rebased onto main.

## Log

- 2026-11-15: filed.
- 2026-11-15 16:36-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15 16:38-07:00: ADR 261115-t9jd (accepted) adds go.yaml.in/yaml/v3. internal/frontmatter parses and edits front matter through the node tree: an unchanged note comes back byte for byte, changed values keep their place and quoting, new keys go at the end, the body is untouched; WriteFile is atomic and keeps the mode. box.Date writes as a plain scalar, and a YAML date with no zone reads as local wall-clock time. A card written through a renaming mapping reads back the same from its file.
