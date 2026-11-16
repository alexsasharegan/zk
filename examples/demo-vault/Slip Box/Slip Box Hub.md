# Slip Box Hub

The entrance to this slip box. Everything an agent needs is here in plain text; the views below are embedded from the base.

## What is here

| What | Where |
|---|---|
| Cards | `cards/zk-<key>.md` |
| Sources | `sources/<slug>.md` |
| The box's spec | `zk.toml` (Obsidian does not show it; open it in a text editor) |
| The base | `Slip Box Base.base` |

A card holds one thought, at a permanent key (`2a1`). Its kind says what sort of statement it is: claim, fact, idea, question, quote, report or hub. Its status says how far along it is: `raw` (captured), `proposed` (placed and linked), `linked` (the owner agrees with its place).

## Working the box

- Make a card: `zk new idea "A thought"`, or `zk new claim "What they said" --under 2`.
- List it: `zk list --tree`; one family: `zk list --under 2`.
- Check it: `zk check`, and `zk fix` for the derived fields.

## Raw

![[Slip Box Base.base#Raw]]

## Proposed

![[Slip Box Base.base#Proposed]]

## All

![[Slip Box Base.base#All]]
