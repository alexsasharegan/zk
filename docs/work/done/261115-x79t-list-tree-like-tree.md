---
id: "261115-x79t"
title: Draw zk list --tree like the tree command
status: done
kind: feature
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
summary: zk list --tree draws the card tree with connector lines, like the tree command, instead of indenting keys.
---

# Draw zk list --tree like the tree command

## Goal

Alexander tried zk on the demo vault (2026-11-15): list --tree only indents keys, which is hard to read once a box is deep. Draw it like the tree command.

Done when:

- [x] zk list --tree draws connector lines (├── └── │) with each card's key, title, kind and status, roots at the left margin.
- [x] With a filter (--status, --kind, --under), a card hangs under its nearest shown ancestor, so the lines stay right.
- [x] --ascii draws the same tree with plain characters (|-- `-- |) for terminals without box-drawing glyphs.

## Next

## Outcome

- Tree drawing: TestDrawTree, TestListTreeCommand in internal/cli/tree_test.go.
- Filters keep true lines: TestDrawTreeFiltered; --tree --under is rooted at the card.
- ASCII: TestDrawTreeASCII.

Decision: none.

## Log

- 2026-11-15: filed.
- 2026-11-15 17:36-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15 17:37-07:00: ticked "zk list --tree draws connector lines (├── └── │) with each card's key, title, kind and status, roots at the left margin.": internal/cli/tree_test.go
- 2026-11-15 17:37-07:00: ticked "With a filter (--status, --kind, --under), a card hangs under its nearest shown ancestor, so the lines stay right.": internal/cli/tree_test.go
- 2026-11-15 17:37-07:00: ticked "--ascii draws the same tree with plain characters (|-- `-- |) for terminals without box-drawing glyphs.": internal/cli/tree_test.go
- 2026-11-15 17:37-07:00: Added internal/cli/tree.go: list --tree draws connector lines with key, title, kind and status; filtered cards hang under their nearest shown ancestor; --ascii for plain characters; with --under, the tree is rooted at that card (plain --under still lists only descendants). Checked on the demo vault and on a scratch copy of the 157-card box.
