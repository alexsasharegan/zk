---
id: "261115-2a31"
title: Check Obsidian's property types, and write them only when asked
summary: zk check compares .obsidian/types.json with the types of zk's fields; --fix-types adds missing entries and never changes one. Read before touching types.json.
status: accepted
date: 2026-11-15
scope: [check, obsidian, internal/box/types.go]
authored_by: agent
---

# Check Obsidian's property types, and write them only when asked

## Context

Obsidian keeps one type per property name for the whole vault, in `.obsidian/types.json`:

```json
{ "types": { "zk_created": "datetime", "zk_level": "number" } }
```

The types are `text`, `multitext`, `number`, `checkbox`, `date`, `datetime`, `aliases` and `tags`. Until a property has an entry, Obsidian guesses, and it guessed `text` for `zk_created` in the demo vault, so dates showed and sorted as text. When the maintainer moved his box into his vault on 2026-11-15 he set the entries by hand.

zk knows every field's type: the spec's mapping (ADR 261115-0tp4) names each field and gives each date field a format. The file is Obsidian's, though, and shared by every note in the vault: a property name the box maps onto may already be used by other notes with another type. Obsidian also holds the file in memory while it runs and may write its own copy over an edit made meanwhile.

## Considered Options

1. **Write `types.json` on every command that opens the box.** Always right, with no step to remember, but zk would rewrite another program's settings unasked, and could change a type other notes rely on.
2. **Only report, in `zk check`, and print the JSON to add by hand.** Never touches the file, but every new vault needs a hand edit.
3. **Report in `zk check`, and write when asked, adding missing entries only.**
4. **Doing nothing.** Each user sets the types in Obsidian's property menu, one by one.

## Decision

Option 3.

The type a field should have comes from the mapping: a date field is `date` or `datetime` by its format, `level` is `number`, `extracted` is `multitext`, and every other field is `text`. Every field the mapping writes is checked, including ones no note uses yet. `aliases` and `tags` belong to Obsidian and are never checked or written.

`zk check` reads `<vault>/.obsidian/types.json` (a missing file has no entries) and reports, with rule `property-type` and the path `.obsidian/types.json`:

- a field with no entry, as a warning (Obsidian guesses, often right); `--strict` makes it an error;
- a field whose entry is another type, as an error, since a value edited in Obsidian would then be written in the wrong shape.

The check writes nothing. Its output names `zk check --fix-types` for missing entries; for a mismatch it says which type the spec wants and leaves the choice to the user, because other notes may use that property.

`zk check --fix-types` adds the missing entries, then checks. It never changes an existing entry, whatever its type, and keeps every other entry and key in the file as it was. It is a flag of its own, not part of `--fix`: `--fix` writes the box's own notes, while this writes Obsidian's settings, and should be a step the user chooses. Because Obsidian may write its in-memory copy back, the output says to reload Obsidian (or close it first).

The writer is a function of the box package that takes a vault and the wanted types, so `zk init` (work item 261115-h674) can set the types for a new box.

## Consequences

- A new vault gets correct types with one command, and an existing one is never changed behind its owner's back.
- `zk check` on a box whose vault has no `types.json` now prints warnings, but passes unless `--strict`.
- A vault where another program already set a zk field to another type fails `zk check` until the owner changes the type in Obsidian or maps the field to another name in `zk.toml`.
- zk writes a file Obsidian owns; if Obsidian changes the file's shape, the reader and writer must follow.

## Relations

- Related to: 261115-0tp4 (the spec's mapping and formats)
