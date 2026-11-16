---
id: "261115-7m9b"
title: Find a box from the vault root
status: raw
kind: feature
created: 2026-11-15
updated: 2026-11-15
---

# Find a box from the vault root

## Goal

ADR 261115-zd6s says "Inside a vault, `zk` works with no flags", but that holds only when the working directory is the box's folder or below it. Discovery walks up from the working directory, so a box in a subfolder of the vault (`Areas/Slip Box/zk.toml`) is not found from the vault root, which is where most people run commands. Today the workaround is `default` in the user config, which is per machine.

Proposed: when the walk up reaches the vault root (the folder holding `.obsidian/`) without finding a spec, look for one inside the vault, and use it when there is exactly one. Open questions for the design:

- How to search: a bounded walk down from the vault root (skipping `.obsidian/`, `.git/`, `.trash/` and dotfolders), or a fixed list of places.
- What to do with two or more boxes in one vault: an error that names them and suggests `--box`, never a silent pick.
- Where it sits in the resolution order: after the walk up and before the user config's `default`, so a box in the vault wins over a default elsewhere.
- Cost on a large vault, and whether the result needs caching (it should not need to).
- `zk config` names the new step ("found in the vault at ...").

Done when:

- [ ] An ADR amends or supersedes 261115-zd6s with the vault search and its place in the order.
- [ ] `zk list`, run at the root of a vault whose only box is in a subfolder, acts on that box, and `zk config` says why.
- [ ] Two boxes in one vault give an error naming both.
- [ ] Tests cover the root, a subfolder outside the box, two boxes, and no box.
- [ ] The README's "Choosing a box" lists the step.

## Next

Raw: Alexander picks when to work it. Start with the ADR draft.

## Log

- 2026-11-15: filed.
