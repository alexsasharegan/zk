---
id: "261115-ahma"
title: Read the box spec and apply the field mapping
status: active
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
kind: feature
summary: Load a box spec file (paths, card kinds, field mapping) and read and write front matter through the mapping, zk_ names by default.
claimed_by: claude-code-opus-5-5
claimed_at: 2026-11-15T23:31
---

# Read the box spec and apply the field mapping

## Goal

Done when:

- [ ] A box spec names the hub, base and the card, source and trail folders, and the card kinds with their descriptions.
- [ ] With no mapping, every field is written as `zk_<name>`; a mapping renames fields and sets value formats (date or datetime); `aliases` and `tags` are never renamed.
- [ ] A missing or unreadable spec is reported plainly, with a hint that Obsidian does not show or by default sync `.toml` files.
- [ ] A mapping that renames `class`, `kind`, `status` and `created` (as a date) round-trips a card and a source in tests.

## Next

Write internal/box: load zk.toml (paths confined to the vault, kinds with defaults, classes) and the field mapping (zk_ defaults, names and date formats), with round-trip tests; then show the spec in zk config.

## Log

- 2026-11-15: filed.
- 2026-11-15 16:31-07:00: claimed (claude-code-opus-5-5).
