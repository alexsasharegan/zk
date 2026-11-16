---
id: "261115-qrhh"
title: Tell Obsidian the types of zk's fields
status: done
kind: feature
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
summary: Obsidian shows zk's dates as text until each property's type is set in the vault; zk should set or check the types of the fields it writes.
---

# Tell Obsidian the types of zk's fields

## Goal

Trying zk on the demo vault (2026-11-15), Obsidian showed `zk_created` as text: it keeps one type per property name per vault, in `.obsidian/types.json`, and guesses until told. zk knows the type of every field it writes, through the spec's mapping and formats.

Done when:

- [x] A decision (an ADR if it rejects a real alternative) on how: zk writes `.obsidian/types.json` entries for its fields (on `zk init`, or a command of its own), or `zk check` reports a mismatch and says how to fix it.
- [x] It never changes the type of a property another note already uses with a different type; it reports that instead.
- [x] The demo vault shows `zk_created` as a date and `zk_level` as a number in Obsidian.

## Next

## Outcome

zk check reports zk's fields whose Obsidian property type is missing (a warning) or another type (an error), and zk check --fix-types adds missing entries without changing any (ADR 261115-2a31). The demo vault ships its types.json; the maintainer saw zk_created as a date and zk_level as a number in Obsidian. box.AddTypes is ready for zk init (261115-h674).

## Log

- 2026-11-15: filed.
- 2026-11-15 22:18-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15: decided with the maintainer (ADR 261115-2a31, proposed): `zk check` reports the box's fields whose type in `.obsidian/types.json` is missing (a warning, an error with `--strict`) or another type (an error), and writes nothing; `zk check --fix-types` adds the missing entries and never changes one. The writer is `box.AddTypes`, for `zk init` to reuse. The demo vault now ships its `types.json`, written by `--fix-types`.
- 2026-11-15 22:38-07:00: ticked "The demo vault shows `zk_created` as a date and `zk_level` as a number in Obsidian.": maintainer's Obsidian screenshots, 2026-11-15: zk_created as date and time, zk_level as number, zk_extracted as a list
