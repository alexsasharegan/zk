---
id: "261115-qrhh"
title: Tell Obsidian the types of zk's fields
status: ready
kind: feature
created: 2026-11-15
updated: 2026-11-15
summary: Obsidian shows zk's dates as text until each property's type is set in the vault; zk should set or check the types of the fields it writes.
---

# Tell Obsidian the types of zk's fields

## Goal

Trying zk on the demo vault (2026-11-15), Obsidian showed `zk_created` as text: it keeps one type per property name per vault, in `.obsidian/types.json`, and guesses until told. zk knows the type of every field it writes, through the spec's mapping and formats.

Done when:

- [ ] A decision (an ADR if it rejects a real alternative) on how: zk writes `.obsidian/types.json` entries for its fields (on `zk init`, or a command of its own), or `zk check` reports a mismatch and says how to fix it.
- [ ] It never changes the type of a property another note already uses with a different type; it reports that instead.
- [ ] The demo vault shows `zk_created` as a date and `zk_level` as a number in Obsidian.

## Next

Read Obsidian's types.json format (the demo vault, after opening it in Obsidian), then propose the approach.

## Log

- 2026-11-15: filed.
