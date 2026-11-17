---
id: "261116-b0w2"
title: Cut the first release, with binaries for macOS and Linux
status: active
kind: feature
created: 2026-11-16
updated: 2026-11-16
summary: Pushing a version tag builds macOS and Linux archives and publishes a GitHub release; the README says how to install one, and v0.1.0 is the first.
claimed_by: claude-code-fable-5-2
claimed_at: 2026-11-17T02:03
---

# Cut the first release, with binaries for macOS and Linux

## Goal

`zk` has no tagged version, so `go install` reports a pseudo-version and there is nothing to download without a Go toolchain. The maintainer wants a real version number and binaries for macOS and Linux, starting with v0.1.0 (ADR 261116-arys, proposed). Windows is left out until someone needs it.

Done when:

- [x] `scripts/release-build.sh <version>` builds one archive per target (macOS and Linux, arm64 and amd64) and a `checksums.txt`, with the version stamped into `zk version`.
- [x] A workflow runs the tests and the script when a version tag is pushed, and publishes a GitHub release with the archives.
- [x] CI runs the script on every change, so the release build cannot rot.
- [x] The README says how to install a binary and how to install with Go, with the Go version `go.mod` asks for, and its status line is current.
- [ ] The maintainer accepts ADR 261116-arys.
- [ ] v0.1.0 is tagged on a green `main`, the release has its four archives, and a downloaded binary prints `zk v0.1.0`.

## Next

The pull request with the script, the workflow and the README is open. After the maintainer merges it and CI on `main` is green, the tag `v0.1.0` is pushed (his call, or on his word) and the release is checked: four archives, checksums, and `zk version` from a downloaded binary. The repository now has an MIT `LICENSE`, which the archives include.

## Log

- 2026-11-16: filed.
- 2026-11-16: the maintainer asked for release mechanics that build binaries for macOS and Linux, and was unsure about Windows. A script in the repository run by a workflow on a tag push was chosen over GoReleaser and over building by hand (ADR 261116-arys, proposed).
- 2026-11-16: built. `scripts/release-build.sh` cross-compiles the four targets with the version set through `-ldflags`, and checks that the binary for the machine it runs on prints it. Run on macOS arm64 for v0.1.0: four archives of 2.5 to 2.9 MB, `checksums.txt` verified with `shasum -c`, the darwin/arm64 binary printed `zk v0.1.0`, and `file` showed a static ELF for linux/amd64. `.github/workflows/release.yml` has not run: it runs only on a tag push. The Linux binaries were not executed.
- 2026-11-16 19:03-07:00: claimed (claude-code-fable-5-2).
- 2026-11-16: the maintainer merged the pull request (19), which accepts ADR 261116-arys, and chose the MIT license, pushed straight to `main` on his word. He asked for `v0.1.0` to be tagged after it.
