# zk

`zk` runs a Zettelkasten slip box inside an Obsidian vault: cards with permanent Luhmann-style keys (`7a1`), the sources they were harvested from, and the hub note and Obsidian base that show them. It has two modes. With full arguments it is a plain command line that scripts and agents can call without ever being prompted; run on a terminal with something missing, it opens a terminal UI to fill in the rest. It is opinionated about Obsidian. For a general plain-text Zettelkasten tool, see [zk-org/zk](https://github.com/zk-org/zk), which shares the name.

Status: early. The first set of commands and the `zk new` UI are done. The records (work, decisions and the roadmap) live in the AKM layout under `docs/`; start with [AGENTS.md](AGENTS.md).

## Install

```sh
go install github.com/alexsasharegan/zk@latest
```

Go 1.24 or newer.

## How a box is laid out

A box is a folder in an Obsidian vault with a spec file, `zk.toml`, in it:

```text
my-vault/
├── .obsidian/
└── Slip Box/
    ├── zk.toml               # the box's spec
    ├── Slip Box Hub.md       # the entrance: what the box is, with the base's views embedded
    ├── Slip Box Base.base    # an Obsidian base over the cards
    ├── cards/zk-2a.md        # one card per file, named for its key
    └── sources/<slug>.md     # one page per thing read
```

A card holds one thought. Its key is its permanent place: `2` is a root, `2a` comments on `2`, `2b` continues `2a`, `2a1` comments on `2a`. A key is never changed or reused. The file is `zk-<key>.md`, and the card carries the tag `zk/2/a`, so the search `tag:#zk/2` finds a card and its whole family.

The words zk uses, the rules for keys and how a box is ordered are in [docs/glossary.md](docs/glossary.md).

Each card has a kind (by default `claim`, `fact`, `idea`, `question`, `quote`, `report` or `hub`) and a status: `raw` (captured), `proposed` (placed and linked, each link with its reason) or `linked` (the owner agrees with its place).

## Try the demo vault

`examples/demo-vault/` is a small, invented box about baking bread. Open it in Obsidian to see the hub and base, or work it from a terminal:

```sh
cd "examples/demo-vault/Slip Box"
zk config              # which box zk chose, and why
zk list --tree         # the cards, drawn as a tree
zk show 2a             # a key's place, and what links to and from it
zk check               # nothing found
zk new idea "Steam makes the crust" --under 2a --dry-run
```

Inside a box, `zk` finds the nearest `zk.toml` by walking up from the working directory, so no setup is needed.

## Commands

| Command | Does |
|---|---|
| `zk new [kind] [title...] [--under KEY]` | Make the next card: a new root, or the next child of KEY. On a terminal, with the kind or title left out, it asks. `--open obsidian\|editor\|none`, `--dry-run`, `--agent` (marks an agent-made card) |
| `zk list` (`ls`) | List cards depth first. `--status`, `--kind`, `--under KEY`; `--tree` draws them like the `tree` command, under the box's name or the `--under` card, with a count (`--ascii` for plain characters) |
| `zk show KEY` | A key's level, sort key, tag and file, and the card's links in and out |
| `zk retitle KEY <title...>` | Change a card's H1 and nickname together. `--links` rewrites old `[[zk-K\|(K) Title]]` links |
| `zk check` | Check cards and sources against the spec, and the property types Obsidian gives the box's fields. `--fix` writes derived fields first; `--fix-types` adds missing types to `.obsidian/types.json` first; `--strict` makes warnings errors |
| `zk fix [KEY...]` | Write derived fields: key, level, sort, tag, and the nickname from the H1 |
| `zk source new <slug> --provenance P` | Make a source page. `--title`, `--author`, `--via`, `--form` |
| `zk source done <slug> KEY...` | Mark a source extracted, with the cards it made |
| `zk config` | The chosen box, why, and what its spec says |

Every listing command takes `--json` and prints one JSON object per line. Exit codes: 0 success, 1 findings or a failed operation, 2 a usage error.

### Writing a card by hand

Run `zk new` on a terminal. It shows the box it chose and why, then asks for a kind (each with what it means), a title, and a parent (`/` searches the cards by key or title; the first entry starts a new root). Esc steps back, ctrl+c cancels. The card then opens in Obsidian, through an `obsidian://` link that launches Obsidian when it is closed; `--open editor` uses `$VISUAL`, else `$EDITOR`, instead. If the card cannot be opened, it is still made, and zk asks whether to open it in your editor. `zk new idea` or `zk new --under 2a` skips the steps already answered.

### For scripts and agents

With every argument given, zk never prompts and opens nothing unless `--open` says so. With an argument missing and no terminal, it exits 2 with the usage and never waits for input. `--no-input`, or `ZK_NO_INPUT=1`, makes a terminal behave the same way. Pass `--agent` to `zk new` when an agent makes a card.

## Configuration

### Choosing a box

zk acts on the first box found by:

1. `--box NAME` (a name from the user config) or `--box PATH` (a spec file or its folder)
2. `ZK_BOX`, read the same way
3. the nearest `zk.toml`, walking up from the working directory and stopping at the home directory
4. `default` in the user config

`zk config` says which step chose the box. When `ZK_BOX` or the default chose it, every command that opens the box also ends with one line on stderr naming it, for example `zk: box notes at ~/notes/Slip Box (the default in ~/.config/zk/config.toml)`. The line is left out when stderr is not a terminal, and with `--json`.

### The user config

The user config only says where boxes are. It is `$ZK_CONFIG`, else `$XDG_CONFIG_HOME/zk/config.toml`, else `~/.config/zk/config.toml`:

```toml
default = "notes"

[boxes.notes]
spec = "~/notes/Slip Box/zk.toml"

[boxes.demo]
spec = "~/src/zk/examples/demo-vault/Slip Box"
```

Nothing in it is merged with a box's spec.

### The box spec

Everything about a box is in its `zk.toml`. Paths are relative to the box's folder and must stay inside the vault.

```toml
name = "Slip Box"
hub = "Slip Box Hub.md"
base = "Slip Box Base.base"
cards = "cards"
sources = "sources"
key_prefix = "zk"           # file zk-7a1.md, tag zk/7/a/1

[kinds.claim]               # leave out [kinds] for the seven defaults
description = "Someone asserts this. Say who, and where."
needs_source = true
```

By default every field zk writes has a `zk_` prefix (`zk_kind`, `zk_status`, `zk_created`), so a box collides with nothing in a vault. If your vault already has fields for some of these, map them:

```toml
[classes]                   # the value that marks a note as a card or a source
card = "zk-card"
source = "zk-source"

[fields]                    # zk's name = your vault's name
class = "type"
kind = "kind"
status = "status"
created = { name = "created_at", format = "date" }
```

The fields zk can map are `class`, `kind`, `key`, `level`, `sort`, `status`, `created` and `authored_by` on cards, and `class`, `title`, `author`, `published`, `provenance`, `via`, `form`, `status`, `created`, `extracted` and `series` on sources. `created` and `published` take a format, `date` or `datetime`. `aliases` and `tags` are Obsidian's own and are never renamed.

### Property types

Obsidian keeps one type per property name for the whole vault, in `.obsidian/types.json`, and guesses until it is told, so a date can show as text. `zk check` compares that file with the types of the box's fields: a date field is `date` or `datetime` by its format, `level` is a number, `extracted` is a list, and the rest are text. A field with no type is a warning; a field set to another type is an error. `zk check` never writes the file.

```sh
zk check --fix-types   # add the missing types, then check
```

It adds missing entries only, and never changes a type that is already set, since other notes in the vault may rely on it. Reload Obsidian afterwards (or run it with Obsidian closed): while Obsidian runs it may write its own copy of the file over the new entries.

## Troubleshooting

**"no spec at ..." on another device.** Obsidian does not show `.toml` files in its file list, and Obsidian Sync copies them only when "Sync other file types" is on in its settings. A box's `zk.toml` can be missing on a phone or a second computer with no sign in the app. Turn that setting on, or copy the file by hand, then run `zk config` to see where zk looked.

**Obsidian opens, but says the vault or file was not found.** Obsidian only opens files in vaults it knows. Open the vault folder in Obsidian once, or use `--open editor`.

**"not inside an Obsidian vault".** zk looks for a `.obsidian/` folder at or above the box's folder. Open the folder in Obsidian once, or create `.obsidian/`.

**"no box found".** Run zk inside a box, pass `--box`, set `ZK_BOX`, or set `default` in the user config.

**A field shows as the wrong type in Obsidian.** Obsidian keeps one type per property name across the whole vault. Run `zk check`: a field with no type is fixed by `zk check --fix-types`. A field set to another type is left for you, because another note may use `kind` or `status` with another meaning: change the type in Obsidian's property menu, keep zk's `zk_` names, or map onto names your vault uses the same way.
