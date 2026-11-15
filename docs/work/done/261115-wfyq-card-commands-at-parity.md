---
id: "261115-wfyq"
title: Port the slip commands to zk
status: done
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
kind: feature
summary: Keys, sort keys and the commands new, list, show, retitle, check (with --fix) and source new and done, at parity with slip.
decisions: [261115-t9jd]
---

# Port the slip commands to zk

## Goal

Done when:

- [x] Key parsing, next sibling and next child, level, sort key and tag match `slip`.
- [x] `new`, `list` (alias `ls`), `show`, `retitle`, `check` (`--fix`, alias `fix`), `source new` and `source done` work with full arguments and never prompt.
- [x] `--json` prints JSON Lines on the listing commands.
- [x] Without a terminal, a missing argument exits non-zero with usage; `--no-input` and `ZK_NO_INPUT` force that on a terminal.

## Next

## Outcome

- Keys at parity with slip: internal/key/key_test.go (sort and level cases, next sibling and child, gaps).
- Commands with full arguments, never prompting: internal/cli/cards_test.go (TestNewMakesCardsInKeyOrder, TestCheckRules, TestFixFinishesAHandMadeCard, TestRetitle, TestSources, TestMappedBoxWritesVaultFields).
- --json as JSON Lines: TestNewOptions, TestCheckCleanBoxAndFindings, TestListAndShow.
- No terminal, missing argument: exit 2 without reading stdin; --no-input and ZK_NO_INPUT force it: TestNewMissingArgumentsNeverPrompt.
- Real data: a scratch copy of the 157-card box checks clean through a mapping, and zk fix changes no card.

Decision: 261115-t9jd (the YAML library). Not here: the terminal UI (261115-ct75) and the parity suite on the demo vault (261115-823p).

## Log

- 2026-11-15: filed.
- 2026-11-15 16:36-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15 16:38-07:00: ADR 261115-t9jd (accepted) adds go.yaml.in/yaml/v3. internal/frontmatter parses and edits front matter through the node tree: an unchanged note comes back byte for byte, changed values keep their place and quoting, new keys go at the end, the body is untouched; WriteFile is atomic and keeps the mode. box.Date writes as a plain scalar, and a YAML date with no zone reads as local wall-clock time. A card written through a renaming mapping reads back the same from its file.
- 2026-11-15 16:47-07:00: ticked "Key parsing, next sibling and next child, level, sort key and tag match `slip`.": internal/key/key_test.go (the slip cases)
- 2026-11-15 16:47-07:00: ticked "`new`, `list` (alias `ls`), `show`, `retitle`, `check` (`--fix`, alias `fix`), `source new` and `source done` work with full arguments and never prompt.": internal/cli/cards_test.go
- 2026-11-15 16:47-07:00: ticked "`--json` prints JSON Lines on the listing commands.": TestNewOptions, TestCheckCleanBoxAndFindings, TestListAndShow in internal/cli/cards_test.go
- 2026-11-15 16:47-07:00: ticked "Without a terminal, a missing argument exits non-zero with usage; `--no-input` and `ZK_NO_INPUT` force that on a terminal.": TestNewMissingArgumentsNeverPrompt, TestUsageErrorsExit2
- 2026-11-15 16:47-07:00: Built internal/key (keys, sort keys, tags, next root and child, as slip), the box store (cards, sources, vault-wide link resolution), check (the slip rules, through the mapping and the spec's kinds) and fix, and the commands new, list/ls, show, retitle, check (--fix, --strict), fix, source new and done, with --json and --no-input / ZK_NO_INPUT. Keys and sort keys are always double-quoted. On a scratch copy of the real 157-card box, with a spec mapped to its current field names, every card passes and zk fix changes nothing; the 19 findings are sources that use captured_at and file.
