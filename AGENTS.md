# zk

A Go command-line tool and terminal UI for running a Zettelkasten slip box in an Obsidian vault; used by Alexander Regan and the agents that help him.

## Commands

- Build: `go build -o zk .`; install: `go install .`.
- Test: `go test ./...`; vet: `go vet ./...`; format: `gofmt -l .` must print nothing.
- Records check: `akmctl check`; `akmctl index` regenerates `docs/adr/README.md`.
- Format records: `npx prettier@<pinned version> --check AGENTS.md "docs/**/*.md"` (pin the version you use).
- Full check before a commit: `gofmt -l .`, `go vet ./...`, `go test ./...`, `akmctl check` and the format command above.

## Where things live

- `docs/adr/`: decisions; `docs/adr/README.md` is the generated routing index.
- `docs/ROADMAP.md`: the order of work. Open work: `akmctl ls --status ready,active,blocked`.
- `docs/work/`: work items; `docs/work/done/` holds closed ones.
- `.agents/skills/`: the procedures (`akm`, `akm-track-work`, `akm-record-decision`).
- `docs/.layout`: the manifest.

## Rules

- Before starting, list the open work. Resume the `active` item you hold; if you hold none, take the top `ready` item; if several `active` items are unclaimed, ask. The `claims` extension is on: use `claim` and `release`. Claim with a name unique to your session, `--as <agent>-<model>`, never a bare `claude` or `codex`, so two sessions never share a claim.
- Work only under a work item. If none fits, create a `raw` one (`akmctl new work <slug>`) and ask which item to work.
- A hard-to-reverse choice, or one that rejects a real alternative, needs an ADR draft with status `proposed`. Never edit the body of an `accepted` ADR; supersede it.
- Update the item's `## Next` and `## Log` in the same commit as the code.
- Run `akmctl check` before committing records.
- An `unknown command` or an unknown flag usually means a stale `akmctl`: run `akmctl version` and update it (`akmctl skills update` refreshes the skills) before reporting a tool bug.
- Code and tests win over prose. When a document is wrong, fix it or say so.
- This file wins over a shared skill where they disagree. The tool never stages or commits; stage by path.
- Without `akmctl`, use the fallbacks below and tell the maintainer to install it.
- Mint ids with `akmctl new`. Without the tool: `id="$(date +%y%m%d)-$(LC_ALL=C tr -dc '0-9a-hjkmnp-tv-z' </dev/urandom | head -c 4)"`, then `ls docs/work docs/work/done docs/adr | grep "^$id-"` must print nothing.
- Without the tool: decisions are in `docs/adr/README.md`; `grep -lE '^status: ?"?active"?$' docs/work/*.md` lists active work (same for `ready`, `blocked`); read an item's `## Next` to resume it.
- Without the tool, create a record by copying the template in `.agents/skills/*/assets/`, naming the file `<id>-<slug>.md` with the same `id`. Claim an item by making the first `## Log` line `claimed (where)`.
- Without the tool, close an item by setting `status`, writing `## Outcome`, setting `updated` and moving it to `docs/work/done/` with `git mv`. Never hand-edit the ADR index.

- **After merging a pull request into `main`**, update the maintainer's installed `zk` as the last step of the merge: bring a checkout of `main` up to date (`git fetch` and `git merge --ff-only origin/main`), then run `go install .` there, and check `zk version` names the new commit. It installs to `$GOBIN`, which is on the maintainer's `PATH`. Do it in the same session as the merge; do not set up a scheduled or background job for it.

## Do not, without asking

- Push a tag, create a release, change repository visibility or settings, or publish.
- Add a dependency.
- Set up a launchd job, cron job, or any other scheduled or background task.
- Put credentials, tokens or private material in any file: the `safety` extension checks known patterns only (add your own to `docs/.akm-deny`).
- Edit the body of an accepted ADR, or hand-edit `docs/adr/README.md`.
