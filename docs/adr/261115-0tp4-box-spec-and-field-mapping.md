---
id: "261115-0tp4"
title: Each box carries its own spec with paths, kinds and a field mapping
summary: The zk.toml spec inside a box, its paths, card kinds and field mapping; zk_ names by default. Read before reading or writing front matter.
status: accepted
date: 2026-11-15
scope: [box spec, front matter, field mapping]
authored_by: agent
---

# Each box carries its own spec with paths, kinds and a field mapping

## Context

A box is a hub note, an Obsidian base and a managed folder of cards and sources. Vaults have their own front matter conventions: the author's main vault keeps a field registry where `type` is the class of every note and `created_at` is always a date, which the original slip schema contradicted. He decided on 2026-11-15 that `zk` writes `zk_` names for everything by default, so a box in a fresh vault collides with nothing, and that a box can opt in to fields a vault already uses. He also flagged that Obsidian does not index or show a `.toml` file, and Obsidian Sync copies one only when "other file types" is on.

## Considered Options

- **Fixed field names in the tool.** Simple, but every vault must adopt the tool's names, which a vault with its own registry cannot do without breaking it.
- **The spec as a Markdown note with front matter.** Obsidian shows and syncs it, but a mapping is structured data, and a strict vault may forbid lists of maps in front matter.
- **A TOML spec file in the box's folder, with `zk_` defaults and an opt-in mapping.** Chosen.

## Decision

- **The spec** is `zk.toml` in the box's managed folder. Paths in it are relative to that folder and must stay inside the vault (the nearest folder above that holds `.obsidian/`).

```toml
name = "Slip Box"
hub = "Slip Box Hub.md"
base = "Slip Box Base.base"
cards = "cards"
sources = "sources"
key_prefix = "zk"            # file zk-7a1.md, tag zk/7/a/1

[kinds.claim]
description = "What a source or person asserts, attributed. It says nothing about whether it holds."
needs_source = true
# idea, fact, question, quote, report, hub likewise

[classes]                    # values of the class field
card = "zk-card"
source = "zk-source"

[fields]                     # canonical name = vault field, or a table with a format
class = "type"
kind = "kind"
status = "status"
created = { name = "created_at", format = "date" }
title = "title"
author = "author"
published = { name = "published", format = "date" }
authored_by = "authored_by"
```

- **Canonical fields**: `class`, `kind`, `key`, `level`, `sort`, `status`, `created`, `authored_by` on cards; `class`, `title`, `author`, `published`, `provenance`, `via`, `form`, `status`, `created`, `extracted`, `series` on sources. A field not in `[fields]` is written as `zk_<canonical>` (`zk_class`, `zk_created`). The default class values are `card` and `source`, and the default `created` format is a datetime, `YYYY-MM-DDTHH:mm:ss`.
- **Never mapped**: `aliases` and `tags` are Obsidian's own fields and keep their names.
- **Kinds** live in the spec, with the description the terminal UI shows and the rule the check applies (`needs_source`, or `needs_link` for hub cards), so a box can add or drop kinds without a new release.
- **A missing spec** is reported plainly: the path looked for, and a note that Obsidian neither shows nor by default syncs `.toml` files, so the file may not have reached this device. The README has a troubleshooting section for it.

## Consequences

- A vault with its own conventions needs a spec that maps to them; the author's main vault keeps its mapping in its own records, outside this repository.
- `zk check` validates front matter through the mapping, so the same code serves a default box and a mapped one.
- Changing a mapping on a full box means rewriting its notes; `zk` does not do that yet. A later command could.
- The spec is invisible in Obsidian, so a user edits it in a text editor.

## Confirmation

- A card and a source written under a mapping that renames `class`, `kind`, `status` and `created` (as a date) carry exactly the mapped names, and read back the same.
- A box with an empty `[fields]` writes only `zk_` names, plus `aliases` and `tags`.

## Review

Revisit if Obsidian gains a way to show or sync `.toml` files, or if a second vault needs value mappings beyond names and date formats.

## Revert

Before release, edit freely. After release, the spec format gets a `version` key, and changes go through a new ADR.
