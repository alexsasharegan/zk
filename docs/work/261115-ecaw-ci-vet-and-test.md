---
id: "261115-ecaw"
title: Run gofmt, vet and tests in CI
status: active
kind: chore
created: 2026-11-15
updated: 2026-11-15
parent: 261115-w9sn
summary: A GitHub Actions workflow that runs gofmt, go vet and go test on pull requests and pushes to main.
claimed_by: claude-code-opus-5-5
claimed_at: 2026-11-15T23:25
---

# Run gofmt, vet and tests in CI

## Goal

Pull requests are checked only on the author's machine. Add CI before the next build items so each PR shows its checks on GitHub.

Done when:

- [x] `.github/workflows/ci.yml` runs `gofmt -l .` (failing on any output), `go vet ./...` and `go test -race ./...` on pull requests and on pushes to `main`, on Linux and macOS, with the Go version from `go.mod`.
- [ ] The workflow passes on its own pull request.

## Next

Read the first CI run on the PR. If it passes, tick box 2 and close this item; if it fails, fix the workflow.

## Log

- 2026-11-15: filed.
- 2026-11-15 16:25-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15 16:26-07:00: ticked "`.github/workflows/ci.yml` runs `gofmt -l .` (failing on any output), `go vet ./...` and `go test -race ./...` on pull requests and on pushes to `main`, on Linux and macOS, with the Go version from `go.mod`.": .github/workflows/ci.yml
- 2026-11-15 16:26-07:00: Wrote .github/workflows/ci.yml: gofmt, go vet, go test -race on ubuntu-latest and macos-latest, Go from go.mod, on pull requests and pushes to main. The same steps pass locally.
