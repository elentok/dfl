# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog.

## [0.2.5] - 2026-04-27

- Added `dfl inject <source-file> <target-file>` for appending managed injected content into target
  files, it uses HTML comment markers for managed blocks, replaces existing injected blocks on
  rerun, and prints explicit step-start output with terse `done` success status.

## [0.2.4] - 2026-04-19

- `dfl setup` now includes component install headers in the final setup summary.

## [0.2.3] - 2026-04-19

- `dfl setup` now prints a final step summary with per-step success/skip lines and detailed failed-step output.
- `dfl git-clone` now inherits GitHub SSH/HTTPS transport from the dotfiles repo when given `owner/repo`, while still preserving explicit clone URLs as-is.
- `dfl git-clone --update` now reports `up-to-date`, `N commits pulled`, or `failed to pull` based on the actual pull result.

## [0.2.2] - 2026-04-18

- `dfl update` now offers to stash tracked local changes before pulling and restores the stash after a successful pull.

## [0.2.1] - 2026-04-18

- Added `dfl pkg github install <owner/repo...>` for installing binaries from GitHub releases.

## [0.2.0] - 2026-04-13

### Added

- Added `dfl update` to self-update the binary, update the target dotfiles repo, and rerun setup.
- Added `--repo` support to `dfl setup` for explicit bootstrap handoff and targeted repo execution.
- Added the public `install-dfl.sh` bootstrap installer shim.

### Changed

- `dfl setup` now runs `core/setup` from the resolved repo root and exports `DFL_COMPONENT_ROOT=<repo>/core`.
- The root `bootstrap` flow now installs `dfl` only when needed and then delegates to `dfl setup --repo <repo>`.
- Setup and component install scripts now get the `dfl` executable prepended to `PATH`, which keeps nested runtime command lookups stable.
- `dfl update --dry-run` now reports the planned setup step without requiring a preinstalled target binary.

## [0.1.0] - 2026-04-11

### Added

- `dfl setup` now runs the repo root `setup` script directly, which makes the current dotfiles setup flow the primary setup entrypoint again.

### Changed

- `dfl setup` now respects the `setup` script shebang instead of forcing `sh`, so Bash-based setup scripts work correctly.
- Component install resolution is now script-only.

### Removed

- Removed the YAML/manifest-based setup implementation and deleted `setup/default.yaml`.
- Removed the YAML dependency and the setup-manifest orchestration code that is no longer used.

## [0.0.3] - 2026-04-10

### Added

- Added interactive runtime commands and git helper commands.

### Changed

- Improved UI and styling.

## [0.0.2] - 2026-04-08

### Changed

- Migrated setup data from TOML to YAML.
- Migrated default manifest package data into the setup flow.
- Cleaned up bootstrap test fixtures after the TOML removal.

## [0.0.1] - 2026-04-07

### Added

- Added the initial bootstrap script.
- Added repo sync support.
- Added setup manifest execution and component manifest support.
- Added CI for tests, releases, and Homebrew packages.
