# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog.

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
