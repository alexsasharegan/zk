---
id: "261115-t9jd"
title: zk reads and writes front matter with go.yaml.in/yaml/v3
summary: Adds go.yaml.in/yaml/v3 to the dependencies of ADR 261115-sbpe, for note front matter. Read before touching front matter parsing.
status: accepted
date: 2026-11-15
scope: [front matter, dependencies]
authored_by: agent
---

# zk reads and writes front matter with go.yaml.in/yaml/v3

## Context

Cards and sources are Markdown notes whose fields are YAML front matter. ADR 261115-0tp4 maps those fields, but reading and writing them in files needs a YAML library, and ADR 261115-sbpe allows no dependency beyond its list without Alexander's word. zk edits notes that people also edit in Obsidian, so a rewrite must keep the key order, the quoting and everything outside the fields zk owns, and leave the note body byte for byte.

## Considered Options

- **`go.yaml.in/yaml/v3`**, the continuation of `gopkg.in/yaml.v3` maintained by the YAML organisation since the original repository was archived. Its node API keeps key order, styles and comments, so an edit can change only the values zk owns. Its error messages are plain.
- **`github.com/goccy/go-yaml`**, actively maintained, with clearer error messages that point at the line and column. Its API is less familiar.
- **A hand-written parser** for the subset Obsidian writes. No dependency, but notes edited by hand can hold any YAML.
- **Doing nothing**, keeping front matter out of zk. The commands cannot work without it.

## Decision

`go.yaml.in/yaml/v3` (Alexander, 2026-11-15, on the agent's recommendation). It is added to the dependency list of ADR 261115-sbpe. Front matter is edited through its node tree, not by decoding into a map and encoding it again.

## Consequences

- Rewriting a note keeps the order and quoting of keys zk does not change; changed keys keep their position; new keys go at the end.
- YAML errors in a note are reported with the library's messages, which name a line but are terser than goccy's. If they prove hard to act on, zk can add the file path and context around them, or this choice can be revisited.

## Confirmation

- `go list -m all` shows `go.yaml.in/yaml/v3` and no other YAML library.
- A test rewrites a card in the shape Obsidian writes and gets the same bytes back when nothing changed.

## Review

Revisit if front matter errors confuse users, or if the library stops being maintained.

## Revert

Replace the package behind `internal/frontmatter`; nothing outside it imports the library.
