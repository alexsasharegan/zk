---
id: "261116-b0w2"
title: Cut the first release, with binaries for macOS and Linux
status: done
kind: feature
created: 2026-11-16
updated: 2026-11-16
closed: 2026-11-16
summary: Pushing a version tag builds macOS and Linux archives and publishes a GitHub release; the README says how to install one, and v0.1.0 is the first.
---

# Cut the first release, with binaries for macOS and Linux

## Goal

`zk` has no tagged version, so `go install` reports a pseudo-version and there is nothing to download without a Go toolchain. The maintainer wants a real version number and binaries for macOS and Linux, starting with v0.1.0 (ADR 261116-arys, proposed). Windows is left out until someone needs it.

Done when:

- [x] `scripts/release-build.sh <version>` builds one archive per target (macOS and Linux, arm64 and amd64) and a `checksums.txt`, with the version stamped into `zk version`.
- [x] A workflow runs the tests and the script when a version tag is pushed, and publishes a GitHub release with the archives.
- [x] CI runs the script on every change, so the release build cannot rot.
- [x] The README says how to install a binary and how to install with Go, with the Go version `go.mod` asks for, and its status line is current.
- [x] The maintainer accepts ADR 261116-arys.
- [x] v0.1.0 is tagged on a green `main`, the release has its four archives, and a downloaded binary prints `zk v0.1.0`.

## Next

## Outcome

v0.1.0 is released: https://github.com/alexsasharegan/zk/releases/tag/v0.1.0. Pushing a version tag runs .github/workflows/release.yml, which tests, builds macOS and Linux archives (arm64 and amd64) with scripts/release-build.sh and publishes them with a checksums file; CI runs the same script on every change. The repository has an MIT license, and the README says how to install a binary or build with Go. Checked on macOS arm64 from the published release: checksums, archive contents and zk version. Not done: the Linux binaries were never executed, the macOS binaries are unsigned, and there is no Windows build, Homebrew formula or install script.

## Log

- 2026-11-16: filed.
- 2026-11-16: the maintainer asked for release mechanics that build binaries for macOS and Linux, and was unsure about Windows. A script in the repository run by a workflow on a tag push was chosen over GoReleaser and over building by hand (ADR 261116-arys, proposed).
- 2026-11-16: built. `scripts/release-build.sh` cross-compiles the four targets with the version set through `-ldflags`, and checks that the binary for the machine it runs on prints it. Run on macOS arm64 for v0.1.0: four archives of 2.5 to 2.9 MB, `checksums.txt` verified with `shasum -c`, the darwin/arm64 binary printed `zk v0.1.0`, and `file` showed a static ELF for linux/amd64. `.github/workflows/release.yml` has not run: it runs only on a tag push. The Linux binaries were not executed.
- 2026-11-16 19:03-07:00: claimed (claude-code-fable-5-2).
- 2026-11-16: the maintainer merged the pull request (19), which accepts ADR 261116-arys, and chose the MIT license, pushed straight to `main` on his word. He asked for `v0.1.0` to be tagged after it.
- 2026-11-16: CI on `main` was green for the license commit (754764c) on Ubuntu and macOS, and `v0.1.0` was tagged there and pushed. The release workflow's first run passed and published the release with four archives and `checksums.txt`. Downloaded all four with `curl`: checksums verified, each archive holds `zk`, `README.md` and `LICENSE`, and the darwin/arm64 binary printed `zk v0.1.0`. The Go proxy resolves `github.com/alexsasharegan/zk@v0.1.0`. The Linux binaries were not executed.
