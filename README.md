# zk

`zk` runs a Zettelkasten slip box inside an Obsidian vault: cards with permanent Luhmann-style keys (`7a1`), the sources they were harvested from, and the hub note and Obsidian base that show them. It has two modes. With full arguments it is a plain command line that scripts and agents can call without ever being prompted; run on a terminal, it opens a terminal UI to fill in what is missing from `zk new`, and to read the box with `zk list`. It is opinionated about Obsidian. For a general plain-text Zettelkasten tool, see [zk-org/zk](https://github.com/zk-org/zk), which shares the name.

Status: early, and used daily on one box. The commands below, `zk init`, and the terminal UI for `zk new` and `zk list` are done. The records (work, decisions and the roadmap) live in the AKM layout under `docs/`; start with [AGENTS.md](AGENTS.md).

## Install

Download a binary for macOS or Linux from the [latest release](https://github.com/alexsasharegan/zk/releases/latest). Each archive is named `zk_<version>_<os>_<arch>.tar.gz`, where `<os>` is `darwin` (macOS) or `linux` and `<arch>` is `arm64` (Apple silicon, ARM) or `amd64` (Intel, AMD). For example, on an Apple silicon Mac:

```sh
curl -fsSLO https://github.com/alexsasharegan/zk/releases/download/v0.1.0/zk_v0.1.0_darwin_arm64.tar.gz
tar -xzf zk_v0.1.0_darwin_arm64.tar.gz
sudo mv zk_v0.1.0_darwin_arm64/zk /usr/local/bin/
zk version
```

`checksums.txt` on the release page lists the SHA-256 of each archive. The macOS binaries are not signed: one downloaded with `curl` runs as it is, but one downloaded with a browser is blocked until you run `xattr -d com.apple.quarantine zk`. There is no Windows build.

Or build it with Go 1.26 or newer:

```sh
go install github.com/alexsasharegan/zk@latest
```

`zk version` prints the version either way.

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

## Start a box

Open the vault in Obsidian once, so it has a `.obsidian/` folder, then:

```sh
cd ~/my-vault
zk init "Slip Box"            # or: zk init "Areas/Notes" --name notes
```

`zk init` writes `zk.toml`, `Slip Box Hub.md` and `Slip Box Base.base`, makes the `cards` and `sources` folders, and sets the property types of zk's fields in Obsidian (reload Obsidian afterwards). It refuses, writing nothing, outside a vault, inside another box or around one, or when one of those files exists, so running it twice is safe. It ends by showing how to name the box in the user config. The new box uses zk's `zk_` field names; to map them onto your vault's own fields, edit `zk.toml` and the base (see "The box spec").

## Commands

| Command                                  | Does                                                                                                                                                                                                                                                                                                      |
| ---------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `zk new [kind] [title...] [--under KEY]` | Make the next card: a new root, or the next child of KEY. On a terminal, with the kind or title left out, it asks. `--open obsidian\|editor\|none`, `--dry-run`, `--agent` (marks an agent-made card)                                                                                                     |
| `zk list` (`ls`)                         | List cards depth first. `--status`, `--kind`, `--under KEY`; `--tree` draws them like the `tree` command, under the box's name or the `--under` card, with a count (`--ascii` for plain characters). On a terminal it opens the interactive list; `--open obsidian\|editor` says where enter opens a card |
| `zk show KEY`                            | A key's level, sort key, tag and file, and the card's links in and out                                                                                                                                                                                                                                    |
| `zk retitle KEY <title...>`              | Change a card's H1 and nickname together. `--links` rewrites old `[[zk-K\|(K) Title]]` links                                                                                                                                                                                                              |
| `zk check`                               | Check cards and sources against the spec, and the property types Obsidian gives the box's fields. `--fix` writes derived fields first; `--fix-types` adds missing types to `.obsidian/types.json` first; `--strict` makes warnings errors                                                                 |
| `zk fix [KEY...]`                        | Write derived fields: key, level, sort, tag, and the nickname from the H1                                                                                                                                                                                                                                 |
| `zk source new <slug> --provenance P`    | Make a source page. `--title`, `--author`, `--via`, `--form`                                                                                                                                                                                                                                              |
| `zk source done <slug> KEY...`           | Mark a source extracted, with the cards it made                                                                                                                                                                                                                                                           |
| `zk init [folder]`                       | Make a new box in a folder of a vault: spec, hub, base, folders and property types. `--name` (default: the folder's name). Never overwrites                                                                                                                                                               |
| `zk config`                              | The chosen box, why, and what its spec says                                                                                                                                                                                                                                                               |

Every listing command takes `--json` and prints one JSON object per line. Exit codes: 0 success, 1 findings or a failed operation, 2 a usage error.

### Writing a card by hand

Run `zk new` on a terminal. It shows the box it chose and why, then asks for a kind (each with what it means), a title, and a parent (`/` searches the cards by key, title, kind or status; the first entry starts a new root). Esc steps back, ctrl+c cancels. The card then opens in Obsidian, through an `obsidian://` link that launches Obsidian when it is closed; `--open editor` uses `$VISUAL`, else `$EDITOR`, instead. If the card cannot be opened, it is still made, and zk asks whether to open it in your editor. `zk new idea` or `zk new --under 2a` skips the steps already answered.

### Reading the box on a terminal

Run `zk list` on a terminal and it opens the interactive list instead of printing: the cards on a full screen, under a header that names the box and why it was chosen. `--status`, `--kind` and `--under` narrow it first, and `--tree` starts it in the tree.

| Key                                              | Does                                                                                                                                                                           |
| ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| up, down, `k`, `j`, page up, page down, `g`, `G` | Move                                                                                                                                                                           |
| `/`                                              | Filter: type to keep the cards whose key, title, kind or status match. Enter keeps the filter, esc clears it                                                                   |
| `t`                                              | Switch between the rows and the tree                                                                                                                                           |
| left or `h`, right or `l`, space                 | In the tree, fold a branch, unfold it, or do whichever applies. A folded card shows how many cards it hides, as `[+3]`; left on a card with nothing to fold goes to its parent |
| shift+left or `H`, shift+right or `L`            | In the tree, fold the whole family the cursor is in and go to its top card; unfold everything under the cursor                                                                 |
| `-`, `+` (or `=`)                                | In the tree, fold every branch, so only the top cards show and each unfold opens one level; unfold every branch                                                                |
| enter                                            | Open the card in Obsidian, or in `$VISUAL`, else `$EDITOR`, with `--open editor`, and come back to the list                                                                    |
| `e`                                              | Open the card in the editor                                                                                                                                                    |
| `q`, esc, ctrl+c                                 | Quit. Esc clears a filter first                                                                                                                                                |

The key is coloured and the title is plain, since those are what you look for; the kind, the status and the tree's lines are faint, and a fold's count stands out. The colours are your terminal's own. The line under the cursor has a `>` and a quiet grey background, with the same colours on it.

If a card cannot be opened, the list says why and offers `e`; nothing falls back on its own. Quitting leaves the screen as it was. The list shows the box as it was when it opened: a title changed in the editor shows the next time you run it.

To get the text on a terminal, pass `--no-input` or pipe it (`zk list | cat`). With no terminal, `--json`, `--no-input` or `ZK_NO_INPUT=1`, `zk list` prints what it always has.

### For scripts and agents

With every argument given, zk never prompts and opens nothing unless `--open` says so. `zk list` opens its interactive list only on a terminal, and never with `--json`. With an argument missing and no terminal, it exits 2 with the usage and never waits for input. `--no-input`, or `ZK_NO_INPUT=1`, makes a terminal behave the same way. Pass `--agent` to `zk new` when an agent makes a card.

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

## Releasing

Pushing a version tag builds the archives and publishes the release (see [the decision](docs/adr/261116-arys-release-binaries.md)):

```sh
git tag v0.1.0 <commit on main>
git push origin v0.1.0
```

A tag with a hyphen, such as `v0.2.0-rc.1`, is published as a prerelease. To build the archives without releasing, run `scripts/release-build.sh v0.1.0`, which writes them to `dist/`. A version cannot be changed once it is published, so fix a bad release with the next version.

## Troubleshooting

**"no spec at ..." on another device.** Obsidian does not show `.toml` files in its file list, and Obsidian Sync copies them only when "Sync other file types" is on in its settings. A box's `zk.toml` can be missing on a phone or a second computer with no sign in the app. Turn that setting on, or copy the file by hand, then run `zk config` to see where zk looked.

**Obsidian opens, but says the vault or file was not found.** Obsidian only opens files in vaults it knows. Open the vault folder in Obsidian once, or use `--open editor`.

**"not inside an Obsidian vault".** zk looks for a `.obsidian/` folder at or above the box's folder. Open the folder in Obsidian once, or create `.obsidian/`.

**"no box found".** Run zk inside a box, pass `--box`, set `ZK_BOX`, or set `default` in the user config.

**A field shows as the wrong type in Obsidian.** Obsidian keeps one type per property name across the whole vault. Run `zk check`: a field with no type is fixed by `zk check --fix-types`. A field set to another type is left for you, because another note may use `kind` or `status` with another meaning: change the type in Obsidian's property menu, keep zk's `zk_` names, or map onto names your vault uses the same way.

## License

[MIT](LICENSE).
