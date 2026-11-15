---
id: "261115-ahma"
title: Read the box spec and apply the field mapping
status: ready
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
kind: feature
summary: Load a box spec file (paths, card kinds, field mapping) and read and write front matter through the mapping, zk_ names by default.
---

# Read the box spec and apply the field mapping

## Goal

Done when:

- [ ] A box spec names the hub, base and the card, source and trail folders, and the card kinds with their descriptions.
- [ ] With no mapping, every field is written as `zk_<name>`; a mapping renames fields and sets value formats (date or datetime); `aliases` and `tags` are never renamed.
- [ ] A missing or unreadable spec is reported plainly, with a hint that Obsidian does not show or by default sync `.toml` files.
- [ ] A mapping that renames `class`, `kind`, `status` and `created` (as a date) round-trips a card and a source in tests.

## Log

- 2026-11-15: filed.
