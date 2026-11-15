---
id: "261115-ahma"
title: Read the box spec and apply the field mapping
status: done
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
kind: feature
summary: Load a box spec file (paths, card kinds, field mapping) and read and write front matter through the mapping, zk_ names by default.
---

# Read the box spec and apply the field mapping

## Goal

Done when:

- [x] A box spec names the hub, base and the card, source and trail folders, and the card kinds with their descriptions.
- [x] With no mapping, every field is written as `zk_<name>`; a mapping renames fields and sets value formats (date or datetime); `aliases` and `tags` are never renamed.
- [x] A missing or unreadable spec is reported plainly, with a hint that Obsidian does not show or by default sync `.toml` files.
- [x] A mapping that renames `class`, `kind`, `status` and `created` (as a date) round-trips a card and a source in tests.

## Next

## Outcome

- Spec paths and kinds: TestLoadDefaults, TestLoadMapped, TestLoadRejects, TestLoadOutsideVault in internal/box/spec_test.go.
- zk_ defaults, renames and formats, aliases and tags untouched: TestDefaultMappingWritesOnlyZkNames, TestMappedCardRoundTrip, TestLoadRejects (map onto tags or aliases).
- Missing or unreadable spec: TestLoadMissingHint, TestResolveMissingSpecHint, TestConfigInvalidSpecFails.
- Round trips under a renaming mapping: TestMappedCardRoundTrip, TestMappedSourceRoundTrip, TestReadsParsedValues.

Not done here: reading and writing front matter in files, which needs a YAML library (not on the ADR 261115-sbpe list); it belongs to 261115-wfyq.

Decision: none (implements ADR 261115-0tp4).

## Log

- 2026-11-15: filed.
- 2026-11-15 16:31-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15 16:34-07:00: ticked "A box spec names the hub, base and the card, source and trail folders, and the card kinds with their descriptions.": internal/box/spec_test.go, internal/box/mapping_test.go, internal/cli/cli_test.go
- 2026-11-15 16:34-07:00: ticked "With no mapping, every field is written as `zk_<name>`; a mapping renames fields and sets value formats (date or datetime); `aliases` and `tags` are never renamed.": internal/box/spec_test.go, internal/box/mapping_test.go, internal/cli/cli_test.go
- 2026-11-15 16:34-07:00: ticked "A missing or unreadable spec is reported plainly, with a hint that Obsidian does not show or by default sync `.toml` files.": internal/box/spec_test.go, internal/box/mapping_test.go, internal/cli/cli_test.go
- 2026-11-15 16:34-07:00: ticked "A mapping that renames `class`, `kind`, `status` and `created` (as a date) round-trips a card and a source in tests.": internal/box/spec_test.go, internal/box/mapping_test.go, internal/cli/cli_test.go
- 2026-11-15 16:34-07:00: Built internal/box: Load reads zk.toml (name required; cards, sources, hub, base relative to the box folder and confined to the vault, found by .obsidian/; key_prefix; kinds replacing seven defaults; unknown keys rejected, inside field tables too). Mapping: zk_ defaults, names and date formats from [fields], class values from [classes], aliases and tags refused; PutCard, Card, PutSource, Source round-trip and leave other fields alone. zk config now loads the spec and prints name, vault, folders and kinds. Front matter files are not read or written yet: that needs a YAML library, which ADR 261115-sbpe does not list.
