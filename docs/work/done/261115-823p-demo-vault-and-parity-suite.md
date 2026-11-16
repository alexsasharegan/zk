---
id: "261115-823p"
title: Build the demo vault and the parity suite
status: done
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
kind: chore
summary: A small public Obsidian vault in the repo that doubles as the test fixture, with the slip test cases ported against it.
---

# Build the demo vault and the parity suite

## Goal

Done when:

- [x] `examples/demo-vault/` is an Obsidian vault with a box spec, hub, base, a few cards and a source, all invented.
- [x] The `slip` test cases are ported and pass against it.
- [x] The README walks through the demo vault, with the troubleshooting note on `.toml` files.

## Next

## Outcome

- Demo vault: examples/demo-vault (spec, hub, base, six cards, one source), all invented; TestDemoVaultIsClean keeps it passing zk check --strict and unchanged by zk fix.
- Parity: TestParity in internal/cli/parity_test.go runs the slip test cases in order against a copy of the demo vault. Not ported, as zk has no such feature: the legacy title migration, kind: research, the ./slip shim, new --help listing kinds.
- README: the demo walkthrough, and the troubleshooting section with the .toml note.

Not checked: the demo hub and base have not been opened in Obsidian.

Decision: none.

## Log

- 2026-11-15: filed.
- 2026-11-15 16:51-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15 16:53-07:00: ticked "`examples/demo-vault/` is an Obsidian vault with a box spec, hub, base, a few cards and a source, all invented.": examples/demo-vault; TestDemoVaultIsClean
- 2026-11-15 16:53-07:00: ticked "The `slip` test cases are ported and pass against it.": TestParity in internal/cli/parity_test.go
- 2026-11-15 16:53-07:00: ticked "The README walks through the demo vault, with the troubleshooting note on `.toml` files.": README.md (demo walkthrough, configuration, troubleshooting)
- 2026-11-15 16:53-07:00: Built examples/demo-vault: an invented box about baking bread (six cards, one of each kind but fact, a source, a hub note and a base), made with zk itself; check --strict is clean. Ported the slip test cases, in order, as TestParity against a copy of it, naming the four that do not apply. Keys in a source's extracted list are now double-quoted like card keys. README rewritten: install, layout, demo walkthrough, commands, configuration and troubleshooting, including the .toml sync note.
