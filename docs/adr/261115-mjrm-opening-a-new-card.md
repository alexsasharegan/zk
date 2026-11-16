---
id: "261115-mjrm"
title: zk new opens the card in Obsidian, or in the editor when asked
summary: "How zk new opens the card it made. Obsidian by default in the terminal UI, --open to choose, and an asked question instead of a silent fallback."
status: accepted
date: 2026-11-15
scope: [zk new, opening notes]
authored_by: agent
---

# zk new opens the card in Obsidian, or in the editor when asked

## Context

After `zk new` makes a card, Alexander wants to write it at once. Most of the time he writes in Obsidian; sometimes in his terminal editor. He asked for both, and asked whether zk should fall back to the editor when Obsidian cannot be opened, or stop and ask.

What can and cannot be detected (macOS; Linux and Windows hand the link to the desktop the same way):

- An `obsidian://open?path=<file>` link launches Obsidian when it is not running, so "Obsidian is closed" is not a failure.
- When Obsidian is not installed, the system has no handler for the link, and the command that opens it exits non-zero. zk can see that.
- When Obsidian opens but the file is in a folder it has never opened as a vault, Obsidian shows its own error. The open command still succeeds, so zk cannot see that.

## Considered Options

1. **Fall back to the editor silently** when the link fails. Convenient, but zk would pick a different place to write than the one asked for, without saying so.
2. **Fail, and leave the card made.** Predictable, but on a terminal it makes the user run a second command.
3. **Fail without a terminal; on a terminal, say why and ask** whether to open the card in the editor instead. Chosen.

## Decision

- `zk new` takes `--open obsidian|editor|none`.
- The default is `obsidian` when the terminal UI made the card, and `none` when every argument was given, so scripts and agents never launch an app.
- `obsidian` opens `obsidian://open?path=<absolute path>` with the system's opener (`open` on macOS, `xdg-open` on Linux, `start` on Windows).
- `editor` runs `$VISUAL`, else `$EDITOR`, on the file, attached to the terminal.
- When opening fails, the card stays made, and zk says so. On a terminal, zk then asks whether to open it in the editor instead (the default answer is no). Without a terminal, it exits 1 with the reason.
- Nothing falls back on its own.

## Consequences

- An unregistered vault shows up as Obsidian's error, not zk's. The README says to open the vault in Obsidian once.
- A default other than Obsidian (a setting in the user config) can be added later without changing the flag.

## Confirmation

- Tests cover each `--open` value, the failure message, the question on a terminal, and exit 1 without one, with the opener replaced by a fake.

## Review

Revisit if Obsidian's command line can report whether a file is in a known vault.

## Revert

Drop the flag; `zk new` prints the path, as before.
