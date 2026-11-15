---
id: "261115-q0r0"
title: Draft the zk design ADRs
status: done
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
kind: spike
summary: Propose the design decisions for zk (CLI shape, config and box resolution, box spec and field mapping, modes) as ADRs for Alexander.
decisions: [261115-sbpe, 261115-zd6s, 261115-0tp4, 261115-czb2]
---

# Draft the zk design ADRs

## Goal

Record the design agreed on 2026-11-15 as proposed ADRs, so the build items have a contract.

Done when:

- [x] Proposed ADRs cover: the CLI framework and command set; the user config and box resolution; the box spec file and field mapping; the non-interactive and interactive modes.
- [x] Alexander has accepted or revised each one.

## Next

## Outcome

- ADRs: docs/adr/261115-sbpe, 261115-zd6s, 261115-0tp4 and 261115-czb2, all accepted.
- Acceptance: Alexander, 2026-11-15.

Decision: 261115-sbpe, 261115-zd6s, 261115-0tp4, 261115-czb2.

## Log

- 2026-11-15: filed.
- 2026-11-15 15:57-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15 15:59-07:00: ticked "Proposed ADRs cover: the CLI framework and command set; the user config and box resolution; the box spec file and field mapping; the non-interactive and interactive modes.": docs/adr/261115-sbpe, 261115-zd6s, 261115-0tp4, 261115-czb2 (proposed)
- 2026-11-15 15:59-07:00: Drafted four proposed ADRs: 261115-sbpe (Cobra, no Viper, command set), 261115-zd6s (user config and box resolution), 261115-0tp4 (box spec and field mapping), 261115-czb2 (prompt only on a terminal with arguments missing).
- 2026-11-15 16:06-07:00: ticked "Alexander has accepted or revised each one.": Alexander accepted all four on 2026-11-15
- 2026-11-15 16:06-07:00: Accepted by Alexander. Before the first public push, references to his private vault and repositories were replaced with general wording.
