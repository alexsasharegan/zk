# zk

`zk` runs a Zettelkasten slip box inside an Obsidian vault: cards with permanent Luhmann-style keys (`7a1`), the sources they were harvested from, and the hub note and Obsidian base that show them. It has two modes. With full arguments it is a plain command line that scripts and agents can call without ever being prompted; run interactively with something missing, it opens a terminal UI to fill in the rest. It is opinionated about Obsidian. For a general plain-text Zettelkasten tool, see [zk-org/zk](https://github.com/zk-org/zk), which shares the name.

Status: design. Nothing is built yet; the decisions are being drafted in `docs/adr/`.

The records (work, decisions and the roadmap) live in the AKM layout under `docs/`; start with [AGENTS.md](AGENTS.md).
