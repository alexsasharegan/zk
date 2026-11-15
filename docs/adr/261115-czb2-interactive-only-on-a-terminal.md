---
id: "261115-czb2"
title: zk prompts only on a terminal with arguments missing
summary: When zk opens its terminal UI and when it must not, so agents and scripts never land in a prompt. Read before adding a prompt.
status: accepted
date: 2026-11-15
scope: [modes, terminal UI]
authored_by: agent
---

# zk prompts only on a terminal with arguments missing

## Context

`zk` serves two users. Alexander wants a terminal UI for writing cards by hand: a scrollable list of kinds with their descriptions, then a title, a parent and an editor. Agents call the same commands and must never be dropped into an interactive prompt they cannot answer. He asked for a bimodal tool with no friction for agents.

## Considered Options

- **A separate subcommand for the UI** (`zk ui new`). Explicit, but two names for one operation.
- **A global `--interactive` flag.** Agents are safe by default, but Alexander types it every time.
- **Prompt only when stdin and stdout are terminals and a required argument is missing.** Chosen.

## Decision

- A command opens its UI only when both stdin and stdout are terminals and a required argument is missing.
- With every required argument given, no command prompts, on a terminal or not.
- Without a terminal, a missing argument exits 2 with the usage line and names what is missing. Nothing waits for input.
- `--no-input`, or `ZK_NO_INPUT=1`, makes a terminal behave as if it were not one.
- Bare `zk` prints help.
- The first command with a UI is `zk new`: kind, then title, then an optional parent found by search, then the card opens in `$EDITOR` or in Obsidian. A flag marks a card an agent made.

## Consequences

- Agents get the plain command line by default, because their shells have no terminal or because they pass every argument.
- A script that runs on a terminal can still pass `--no-input` to be safe.
- Each command with a UI needs a test for both paths.

## Confirmation

- A test runs every command with missing arguments and no terminal, and checks it exits 2 without reading stdin.
- The agent skill for the box tells agents to pass every argument and `--no-input`.

## Review

Revisit when a second command gets a UI, or when bare `zk` becomes a dashboard.

## Revert

Removing the UI leaves the command line unchanged.
