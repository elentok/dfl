# DFL Implementation Plan

## Goal

Build the first working `dfl` milestone as a Go replacement for the current setup/install runtime,
while keeping risk low by preserving existing shell installers and using `framework.sh` as a
compatibility layer during migration.

## Principles

- [ ] Keep milestone 1 behavior close to the current dotfiles flow unless the spec now says
      otherwise.
- [ ] Prefer compatibility shims over repo-wide installer rewrites.
- [ ] Make each step independently reviewable and verifiable before moving on.
- [ ] Keep package policy in manifests and package execution in `dfl pkg ...`.

## Step 1: Establish runtime skeleton

- [x] Create the Go CLI entrypoint and top-level command structure for `dfl`.
- [x] Add command groups for `setup`, `install`, `pkg`, `os`, and the initial filesystem helpers.
- [x] Define shared runtime types for structured operation results: `success`, `skipped`, `failed`.
- [x] Define shared context types for repo root, component info, OS detection, and dry-run mode.
- [x] Implement repo-root discovery rules and confirm they work from nested working directories.
- [x] Add a minimal smoke test covering CLI startup and repo-root resolution.

## Step 2: Implement component discovery and install execution

- [x] Implement component resolution for `core/<name>/install.toml`, `core/<name>/install`,
      `extra/<name>/install.toml`, and `extra/<name>/install`.
- [x] Return resolved component metadata: name, kind, root, installer type, entrypoint.
- [x] Implement `dfl install <component...>` with per-component execution and summary reporting.
- [x] Export the required script environment variables:
      `DFL_ROOT`, `DFL_COMPONENT_ROOT`, and `DOTF`.
- [x] Execute shell installers from the component root.
- [x] Keep the initial default failure policy as stop-on-first-failure.
- [x] Add tests for successful resolution, missing components, and manifest-vs-script precedence.

## Step 3: Implement core runtime commands

- [x] Implement `dfl os is-mac`, `dfl os is-linux`, `dfl os is-wsl`, and `dfl has-command`.
- [x] Implement `dfl step-start` and `dfl step-end`.
- [x] Implement `dfl shell <name> -- <command...>` with streamed output and exit-code propagation.
- [x] Implement `dfl symlink`, `dfl copy`, `dfl mkdir`, and `dfl backup`.
- [x] Implement backup behavior as `<target>.backup` first, then timestamped fallback on
      collision.
- [x] Implement dry-run-aware behavior for all of the above.
- [x] Add focused tests for no-op/skip behavior, backup naming, and dry-run output.

## Step 4: Implement package execution layer

- [x] Implement `dfl pkg brew install <pkg...>`.
- [x] Implement `dfl pkg apt install <pkg...>`.
- [x] Implement `dfl pkg npm install <pkg...>`.
- [x] Implement `dfl pkg pipx install <pkg...>`.
- [x] Implement `dfl pkg cargo install <pkg...>`.
- [x] Implement `dfl pkg snap install <pkg...>`.
- [x] Define per-manager “already installed” checks where practical; fall back to direct install
      when detection is messy.
- [x] Add Homebrew support for ensuring taps before package install when requested by the manifest
      layer.
- [x] Add tests around argument shaping and manifest-to-runtime translation where unit-testable.

## Step 5: Implement manifest parsing

- [x] Define Go structs for `install.toml` and `setup/default.toml`.
- [x] Parse `[when]`, `[symlinks]`, `[copies]`, `mkdirs`, `[[packages]]`, and `[[steps]]`.
- [x] Support `[[packages]]` fields:
      `manager`, `names`, optional `tap`, optional `cask`, `when_os`,
      `when_linux_distro`, and `when_features`.
- [x] Support `[[steps]]` fields:
      `name`, `os`, `if`, `if_not`, `cwd`, `sudo`, and `run`.
- [x] Define machine-context evaluation for OS, Linux distro, and feature tags.
- [x] Add validation errors for malformed manifests and unsupported manager names.
- [x] Add tests for parsing, condition filtering, and invalid manifest cases.

## Step 6: Implement `setup/default.toml` execution

- [x] Define setup-manifest support for:
      `[repo_defaults]`, `[[components]]`, `[when]`, `[[packages]]`, `[[repos]]`, and `[[steps]]`.
- [x] Implement `dfl setup` to load `setup/default.toml`.
- [x] Execute setup manifests in a clear order:
      validate setup `[when]`, sync setup repos, install setup packages, run setup
      filesystem/actions, install selected components, run setup steps that belong after
      components if the final spec keeps that split.
- [x] Support `--component <name>` as a filter over the setup-manifest component entries.
- [x] Support `--skip-packages`.
- [x] Support `--skip-repos` as a skip for the setup-manifest `[[repos]]` phase.
- [x] Add tests for component filtering, repo filtering, dry-run behavior, and setup-level
      conditional execution.

## Step 7: Implement repo synchronization

- [x] Define setup-manifest repo entries with `name`, `path`, and either `github` or `url`.
- [x] Define `[repo_defaults].transport` and per-repo `transport` overrides.
- [x] Implement transport inheritance from the dotfiles repo `origin` remote, with HTTPS fallback
      when the origin is not a clear GitHub SSH or HTTPS remote.
- [x] Implement GitHub URL expansion for `github = "owner/name"` using SSH or HTTPS transport.
- [x] Clone repos when the target path is missing.
- [x] Run `git pull --ff-only` when the target path already exists and is a Git checkout.
- [x] Detect and report diverged branches as a failed repo result with a clear explanation.
- [x] Add dry-run reporting for clone, pull, skip, and failure-precondition cases.
- [x] Add tests for transport resolution, HTTPS fallback, clone-vs-pull behavior, and divergence
      handling.

## Step 8: Migrate current package and setup data

- [ ] Convert `/Users/david/.dotfiles/config/packages.cfg` into `setup/default.toml`
      `[[packages]]` entries.
- [ ] Normalize manager/platform variants from the old file into conditional TOML entries.
- [ ] Represent `dff` as a Homebrew package entry with its required tap.
- [ ] Move repo definitions out of `dotf-repos` and into `setup/default.toml` `[[repos]]`
      entries.
- [ ] Use inherited transport by default and explicit per-repo overrides only where needed.
- [ ] Move repo-level actions such as Deno caching into `setup/default.toml` `[[steps]]`.
- [ ] Represent macOS-only setup actions in the setup manifest using conditions.
- [ ] Keep `osx-tuning` as a component referenced by a conditional setup-manifest component entry,
      not as a special-case built-in action.
- [ ] Review the resulting manifest for readability before wiring it into the default flow.

## Step 9: Add compatibility layer in `framework.sh`

- [ ] Identify the `framework.sh` helpers currently used by install scripts.
- [ ] Replace or wrap the highest-value helpers so they delegate to `dfl` instead of reimplementing
      logic in shell.
- [ ] Start with helpers that map directly to the new runtime:
      symlinking, copying, mkdir, OS predicates, command checks, shell steps, and package installs.
- [ ] Preserve existing script call sites so migrated and non-migrated installers continue to work.
- [ ] Add a small compatibility test surface where practical, or at minimum document the mapped
      helper-to-command behavior.

## Step 10: Migrate a small set of installers directly to `dfl`

- [ ] Pick 2-3 representative installers for early direct migration, such as `core/tmux`,
      `extra/ssh`, and one package-heavy component.
- [ ] Convert only those installers that materially validate the new runtime.
- [ ] Keep complex installers like `core/nvim` shell-driven for now.
- [ ] Verify that direct `dfl` calls and `framework.sh` compatibility wrappers coexist cleanly.
- [ ] Use these migrations to refine output formatting and runtime ergonomics before broader
      adoption.

## Step 11: Verification and cutover

- [ ] Compare `dfl setup` behavior against the current `dotf-setup` on at least one macOS path and
      one Linux path where feasible.
- [ ] Compare `dfl` repo synchronization behavior against current `dotf-repos` expectations, with
      special attention to SSH/HTTPS transport inheritance.
- [ ] Run dry-run checks for setup and install flows.
- [ ] Verify component install summaries and failure handling.
- [ ] Verify package resolution across OS/distro/feature conditions.
- [ ] Verify repo clone, ff-only pull, and divergence reporting behavior.
- [ ] Verify backup collision handling.
- [ ] Verify the compatibility layer still supports legacy install scripts.
- [ ] Document known gaps that remain intentionally deferred to later milestones.

## Review cadence

- [ ] Review after Step 1 before moving to Step 2.
- [ ] Review after Step 3 before moving to package and manifest work.
- [ ] Review after Step 6 before migrating real setup data.
- [ ] Review after Step 9 before direct installer rewrites.
- [ ] Review after Step 11 before considering milestone 2 manifest migration work.
