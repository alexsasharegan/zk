---
id: "261115-ytjf"
title: Build the CLI skeleton, user config and box resolution
status: done
parent: 261115-w9sn
created: 2026-11-15
updated: 2026-11-15
closed: 2026-11-15
kind: feature
summary: Cobra app with the command tree, TOML user config at the XDG path, box resolution order, and zk config to explain the choice.
---

# Build the CLI skeleton, user config and box resolution

## Goal

Done when:

- [x] `zk --help`, `zk version` and `zk completion` work.
- [x] The user config is read from `$ZK_CONFIG`, `$XDG_CONFIG_HOME/zk/config.toml` or `~/.config/zk/config.toml`, never merged.
- [x] A box is chosen by `--box`, `ZK_BOX`, the nearest spec walking up from the working directory (stopping at `$HOME`), then the default; `zk config` prints the box and the reason.
- [x] Tests cover each step of the resolution order.

## Next

## Outcome

- Help, version and completion: TestBareZkPrintsHelp, TestCommandSet, TestVersion, TestCompletion in internal/cli/cli_test.go.
- User config location and no merging: TestUserPathOrder, TestUserPathXDGMissingFallsThrough, TestLoadUser, TestLoadUserRejects in internal/config/config_test.go.
- Resolution order and zk config: TestResolveOrder, TestResolveFlagPath, TestDiscoverStopsAtHome, TestConfigReportsDiscoveredBox, TestConfigBoxFlagByName.
- Exit codes: TestUsageErrorsExit2, TestConfigNoBoxFails.

Decision: none (implements ADRs 261115-sbpe and 261115-zd6s).

## Log

- 2026-11-15: filed.
- 2026-11-15 16:20-07:00: claimed (claude-code-opus-5-5).
- 2026-11-15 16:23-07:00: ticked "`zk --help`, `zk version` and `zk completion` work.": internal/cli/cli_test.go, internal/config/config_test.go (go test ./... passes)
- 2026-11-15 16:23-07:00: ticked "The user config is read from `$ZK_CONFIG`, `$XDG_CONFIG_HOME/zk/config.toml` or `~/.config/zk/config.toml`, never merged.": internal/cli/cli_test.go, internal/config/config_test.go (go test ./... passes)
- 2026-11-15 16:23-07:00: ticked "A box is chosen by `--box`, `ZK_BOX`, the nearest spec walking up from the working directory (stopping at `$HOME`), then the default; `zk config` prints the box and the reason.": internal/cli/cli_test.go, internal/config/config_test.go (go test ./... passes)
- 2026-11-15 16:23-07:00: ticked "Tests cover each step of the resolution order.": internal/cli/cli_test.go, internal/config/config_test.go (go test ./... passes)
- 2026-11-15 16:23-07:00: Built the Go module: Cobra root (bare zk prints help), version (--version too), completion, and zk config with --json; internal/config reads the user config (ZK_CONFIG, XDG, ~/.config; unknown keys rejected) and resolves the box in the ADR 261115-zd6s order. Usage errors exit 2, failures 1. go vet and go test pass.
