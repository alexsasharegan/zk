---
id: "261116-arys"
title: Releases are built by a GitHub workflow when a version tag is pushed
summary: Pushing a version tag runs a workflow that builds macOS and Linux archives with a script in the repository and publishes a GitHub release.
status: proposed
date: 2026-11-16
scope: [release, .github/workflows, scripts/release-build.sh, install]
authored_by: agent
---

# Releases are built by a GitHub workflow when a version tag is pushed

## Context

`zk` has no tagged version. `go install github.com/alexsasharegan/zk@latest` works, but it needs a Go toolchain and reports a pseudo-version (`v0.0.0-<date>-<commit>`). The maintainer wants a real version number, and binaries that people on macOS and Linux can download without Go. He is not sure Windows is needed.

`zk version` already prefers a value set at build time (`-ldflags "-X .../internal/cli.version=..."`) over the module version that `go install` records, so a build can be stamped without a code change. `zk` has no cgo, so every target cross-compiles from one machine. `AGENTS.md` reserves pushing a tag and creating a release for the maintainer.

## Considered Options

1. **A shell script in the repository, run by a workflow on a tag push.** `scripts/release-build.sh` cross-compiles each target, makes one `tar.gz` per target and a `checksums.txt`; the workflow runs the tests, runs the script and publishes with `gh release create`. Nothing new to install or trust, and the build runs the same on a laptop. The script is ours to maintain.
2. **GoReleaser.** The usual tool for Go releases: archives, checksums, a changelog, and later a Homebrew tap or signing from one config file. It adds a third-party action that runs with a token that can write to the repository, and a config format to learn, for four archives.
3. **Build and upload by hand from the maintainer's machine.** No workflow, but each release depends on one machine's Go version and on remembering every step.
4. **Doing nothing: `go install` only.** A tag alone gives a real version number. People without Go cannot install `zk`.

## Decision

Option 1.

- **Trigger.** Pushing a tag that looks like a version (`v0.1.0`, `v0.1.0-rc.1`) runs `.github/workflows/release.yml`. The tag is the release decision; the workflow does the rest. A tag with a hyphen is published as a prerelease.
- **Targets.** `darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`. No Windows: `zk` is used in a terminal beside Obsidian on macOS and Linux today, and nobody has asked. Adding it is one more entry in the script's target list, plus a `.zip`.
- **Archives.** `zk_<version>_<os>_<arch>.tar.gz`, holding a folder of the same name with `zk` and `README.md` (and `LICENSE` once the repository has one). `checksums.txt` lists the SHA-256 of each archive.
- **Version.** The script stamps the tag into the binary with `-ldflags`, and fails if the binary built for the machine it runs on does not print that version.
- **Checks.** The release workflow runs `go vet` and `go test -race` before building. CI runs the script on every change (as `v0.0.0-ci`, publishing nothing), so a broken release build shows up in a pull request and not on release day.
- **Notes.** GitHub generates the release notes from the merged pull requests.

## Consequences

- Cutting a release is: make sure `main` is green, then `git tag v0.1.0 <commit>` and `git push origin v0.1.0`. The maintainer does this, or asks an agent to.
- A version, once fetched through the Go module proxy, cannot be changed. A bad release is fixed by the next version, not by moving the tag. If the workflow fails after the tag is pushed, re-run the workflow; do not delete and re-push the tag.
- The macOS binaries are not signed or notarized. One downloaded with `curl` runs; one downloaded with a browser is quarantined and macOS refuses to open it until the quarantine attribute is removed. The README says so. Signing needs an Apple developer account and is a separate decision.
- The workflow has `contents: write`, which it needs to create the release. It runs only on a tag push, never on a pull request.
- The repository has no `LICENSE` file. That is the maintainer's choice to make, and worth making before the first release.
- No Homebrew formula, no install script and no Windows build. Each can be added later without changing this decision.

## Confirmation

CI's `release build` step runs `scripts/release-build.sh`, which fails when a target does not compile or the stamped version is wrong.

## Revert

Delete `.github/workflows/release.yml` and the CI step. Releases already published stay.

## Relations

- Related to: 261115-sbpe (the CLI and its commands, including `zk version`)
