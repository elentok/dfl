# DFL Spec

This document defines the first concrete design for `dfl`, the Go-based replacement for the current
`framework.sh` runtime in ~/.dotfiles and the `dotf-*` setup/install entrypoints.

The goals are:

- Replace `framework.sh` and the sourced shell helper libraries with a single binary.
- Replace `dotf-setup` with `dfl setup`.
- Replace `dotf-component` with `dfl install`.
- Preserve the current shell `install` scripts during the first migration.
- Support a future migration from `install` scripts to `install.toml` for simple components.

This is intentionally scoped for the first implementation milestone. It does
not attempt to redesign every existing helper script in the repo.

## Notes from the TypeScript experiment

The file [core/fzf/install.ts](/Users/david/.dotfiles/core/fzf/install.ts) and
the related files under [core/framework](/Users/david/.dotfiles/core/framework)
were an earlier attempt to move this runtime into Deno/TypeScript.

That implementation should not be copied directly, but it contains two important ideas worth
preserving in `dfl`:

- a step should produce structured results, not just print ad hoc shell output
- idempotent no-op cases should be a first-class outcome, distinct from ordinary success

In the TypeScript version this showed up as `success`, `silent-success`, and `error`, plus nested
step items. In `dfl`, the naming can be simpler, but the model should survive:

- `success`
- `skipped`
- `failed`

And each top-level operation should be able to include child operations or messages.

## Non-goals

- Replace every existing shell utility under `core/scripts` or `extra/scripts`.
- Express complex installation logic purely in TOML.
- Remove shell scripts immediately.
- Build a general-purpose package manager.

## Terms

- `repo root`: the dotfiles repository root.
- `component`: an installable unit under `core/`, `extra/`, or a future plugin location.
- `runtime command`: a `dfl` subcommand used by component installers, replacing a shell helper.
- `manifest`: an `install.toml` file describing a component declaratively.

## User-facing CLI

### `dfl setup`

Installs the default machine setup.

Responsibilities:

- Read and execute the repo-level setup manifest at `setup/default.toml`.
- Install the default component set declared there.
- Run repo-level package/bootstrap steps declared there.
- Print a final summary including failures and skipped items.

This is the repo-wide orchestration entrypoint. It is intentionally separate from component
manifests because some setup tasks belong to the repo as a whole rather than to a single component.

Suggested flags:

- `--dry-run`
- `--verbose`
- `--component <name>` to limit setup to selected core components
- `--skip-packages`
- `--skip-repos`

### `dfl install <component...>`

Installs one or more components.

Examples:

```bash
dfl install tmux
dfl install fish nvim git
dfl i wezterm
```

Responsibilities:

- Resolve each component by name.
- Load `install.toml` if present, otherwise execute `install`.
- Print a per-component summary.

Suggested aliases:

- `dfl i`

Suggested flags:

- `--dry-run`
- `--verbose`
- `--force`

### `dfl list`

Lists available components.

Suggested output fields:

- component name
- source set: `core` or `extra`
- installer type: `manifest` or `script`
- supported platforms if declared

Suggested flags:

- `--core`
- `--extra`

For milestone 1, machine-readable structured output should remain an internal implementation detail.
Public `--json` output can be added later if a real use case appears.

### `dfl component path <name>`

Prints the absolute path to the resolved component root.

This is primarily for debugging, scripting, and future editor integration.

### `dfl component info <name>`

Prints information about a component:

- resolved path
- installer type
- entrypoint
- supported OS values if declared

### `dfl doctor`

Runs basic environment checks:

- repo root found
- expected package managers available for the current platform
- writable config directories available

This is optional for milestone 1, but the command name should be reserved.

## Component discovery

`dfl` should replace the resolution logic in `core/scripts/dotf-component`.

Resolution order:

1. `core/<name>/install.toml`
2. `core/<name>/install`
3. `core/<name>` if the component is a single executable file
4. `extra/<name>/install.toml`
5. `extra/<name>/install`
6. `extra/<name>`
7. future plugin paths

For milestone 1, supported component entrypoints should be intentionally narrow:

- `install.toml`
- `install`

No `install.*` support should be added. Existing leftovers such as `install.py` or the TypeScript
experiment in `core/fzf/install.ts` should be replaced by either:

- `install.toml` for declarative/simple components
- `install` for shell-driven/imperative components

Each resolved component has:

- `Name`
- `Kind`: `core` or `extra`
- `Root`
- `InstallerType`: `manifest` or `script`
- `Entrypoint`

## Runtime environment for component scripts

When `dfl install <component>` executes a shell installer, it should provide a stable execution
environment.

These variables should be exported into the environment of the executed `install` script, so the
script can read them directly.

Required environment variables:

- `DFL_ROOT`: absolute repo root
- `DFL_COMPONENT_ROOT`: absolute component directory

Compatibility aliases for migration:

- `DOTF=$DFL_ROOT`

Optional future environment variables:

- `DFL_COMPONENT`: resolved component name
- `DFL_COMPONENT_KIND`: `core` or `extra`
- `DFL_OS`: `mac`, `linux`, or `wsl`

The installer working directory should be the component root.

This allows a component script to use short relative paths:

```bash
dfl symlink tmux.conf ~/.tmux.conf
```

instead of:

```bash
dotf-symlink "$DOTF/core/tmux/tmux.conf" ~/.tmux.conf
```

## Runtime commands

These commands replace the subset of `framework.sh` and helper libraries that are actually part of
component installation.

Internally, each runtime operation should return a structured result object with:

- operation name
- status: `success`, `skipped`, or `failed`
- optional message
- optional child results

The first implementation only needs to print these results consistently, but the runtime should be
designed around structured operation results rather than direct printing from every helper. This
structured result model stays internal for now; it should not drive a public JSON CLI surface in
milestone 1.

### Filesystem

#### `dfl symlink <source> <target>`

Creates or updates a symlink.

Behavior:

- `source` is resolved relative to `DFL_COMPONENT_ROOT` if not absolute.
- if `target` already exists and is the correct symlink, print `skip`
- if `target` exists and differs, move it to `<target>.backup`
- if regular user permissions fail, retry with `sudo`

This replaces `dotf-symlink`.

#### `dfl copy <source> <target>`

Copies a file.

Behavior:

- resolve relative source from `DFL_COMPONENT_ROOT`
- skip if target exists with identical contents
- otherwise back up target and replace it
- optionally retry with `sudo`

This replaces `copy_to_dir` and part of `sudo_copy`, but with a single clearer command.

#### `dfl mkdir <path>`

Creates a directory if missing.

Behavior:

- create parents
- skip if directory already exists
- retry with `sudo` when needed

This replaces `make_dir`.

#### `dfl backup <path>`

Moves a path to `<path>.backup`, using `sudo` if required.

This replaces `dotf-backup`.

### Steps and output

#### `dfl step-start <message>`

Starts a step with consistent UI formatting.

#### `dfl step-end --success|--skip|--error [message]`

Finishes the current step.

This replaces the common `dotf-bullet` + `dotf-info` + `dotf-success` pattern and gives scripts a
structured output model.

`--skip` should be used for idempotent no-op cases.

### Shell execution

#### `dfl shell <name> -- <command...>`

Runs a command with standard step formatting.

Example:

```bash
dfl shell "Restoring Neovim plugins" -- nvim --headless "+Lazy! restore" +qa
```

Behavior:

- show a named step
- stream command output
- return the command exit status

This is the preferred runtime wrapper for simple imperative commands.

#### `dfl sudo -- <command...>`

Executes a command via `sudo`.

This is not a formatting helper. It exists to keep shell installers concise and consistent.

### OS and command checks

#### `dfl has-command <name>`

Exits `0` if a command exists.

#### `dfl os is-mac`

#### `dfl os is-linux`

#### `dfl os is-wsl`

Small predicate helpers for shell scripts.

### Packages

For milestone 1, package management should support the operations already used by the repo
setup/install flow. This includes current package-manager variants such as platform-specific brew
and apt groups.

- `dfl pkg brew install <pkg...>`
- `dfl pkg apt install <pkg...>`
- `dfl pkg npm install <pkg...>`
- `dfl pkg pipx install <pkg...>`
- `dfl pkg cargo install <pkg...>`
- `dfl pkg snap install <pkg...>`

Behavior:

- install only missing packages where practical
- print clear skip/install output
- preserve current platform-specific behavior where needed

`dfl pkg ...` is the low-level execution layer. Repo-level policy such as OS gating, distro gating,
desktop-feature gating, taps, or casks should be expressed in manifests and resolved before the
runtime reaches `dfl pkg`.

Examples of manifest-level package conditions:

- Homebrew package available on all brew-capable systems, but only needed on macOS
- APT package that differs between Debian and Ubuntu
- GUI-only or KDE-only package groups

The package-manager-specific implementation details can remain simple at first. The contract
matters more than perfect parity with every edge case in the old shell scripts.

### User, group, and service operations

These are needed for components like `extra/minecraft`.

- `dfl user exists <name>`
- `dfl group exists <name>`
- `dfl group add-user <group> [user]`
- `dfl service start <name>`
- `dfl service stop <name>`

These can be added after the core runtime commands if implementation order requires it, but the
names should be reserved in the spec now.

### Variables

The current `framework.sh` stores machine-specific exported variables in `~/.config/machine`.

Reserve this capability as:

- `dfl var get <key>`
- `dfl var set <key> <value>`

Milestone 1 can defer implementation unless a migrated component depends on it.

## `install.toml` format

The manifest format exists for simple and medium-complexity components.

Complex logic should remain in shell scripts.

Allowed:

- symlinks
- copies
- directory creation
- package installation
- short command execution
- OS gating
- step ordering

Not recommended:

- loops
- branching beyond simple `if`/`if_not` command predicates
- embedded long shell scripts

If a manifest needs more than a few imperative commands, the component should keep an `install`
script and optionally use helper scripts alongside it.

### Example

```toml
name = "tmux"
kind = "core"

[when]
os = ["mac", "linux"]

[symlinks]
"tmux.conf" = "~/.tmux.conf"

[[steps]]
name = "tmux-256color"
os = ["mac"]
if_not = "infocmp tmux-256color >/dev/null 2>&1"
run = "./install-tmux-256color"
```

### Top-level fields

#### `name`

Optional but recommended. Defaults to the resolved component directory name.

#### `kind`

Optional but recommended. Expected values:

- `core`
- `extra`

For milestone 1 this is informational and validated against the resolved path.

#### `[when]`

Optional component-level constraints.

Supported fields:

- `os = ["mac", "linux", "wsl"]`

If the current OS does not match, the component is skipped cleanly.

### Declarative operations

#### `[symlinks]`

String-to-string map.

Rules:

- key is source path relative to component root unless absolute
- value is target path

Example:

```toml
[symlinks]
"config" = "~/.config/ghostty"
"zshrc" = "~/.zshrc"
```

#### `[copies]`

String-to-string map.

Rules match `[symlinks]` but perform file copies instead.

#### `mkdirs`

Array of paths or a table representation if options are added later.

Initial simple form:

```toml
mkdirs = [
  "~/.config",
  "~/.local/share/applications",
]
```

### Packages

#### `[[packages]]`

Ordered package groups with optional conditions.

Example:

```toml
[[packages]]
manager = "brew"
names = ["tmux", "reattach-to-user-namespace"]

[[packages]]
manager = "apt"
names = ["tmux"]

[[packages]]
manager = "cargo"
names = ["eza"]
```

Supported fields:

- `manager`: required, for example `brew`, `apt`, `npm`, `pipx`, `cargo`, `snap`
- `names`: required array of package names
- `tap`: optional Homebrew tap to ensure before installing the listed packages
- `cask`: optional boolean for Homebrew cask installs
- `when_os`: optional list of `mac`, `linux`, `wsl`
- `when_linux_distro`: optional list such as `debian`, `ubuntu`
- `when_features`: optional feature tags such as `gui` or `kde`

Use `names = [...]` even for a single package. This keeps the schema uniform and makes it easy to
group packages that share the same conditions.

The executor installs only package groups that match the current machine context.

### Imperative steps

#### `[[steps]]`

Ordered actions executed after declarative operations unless the implementation explicitly documents
a different order.

Supported fields:

- `name`: required
- `os`: optional list
- `if`: optional shell predicate command
- `if_not`: optional shell predicate command
- `cwd`: optional working directory, relative to component root unless absolute
- `sudo`: optional boolean
- `run`: command string for short commands

Examples:

```toml
[[steps]]
name = "restore plugins"
run = "nvim --headless '+Lazy! restore' +qa"

[[steps]]
name = "setup terminfo"
os = ["mac"]
if_not = "infocmp tmux-256color >/dev/null 2>&1"
run = "./install-tmux-256color"
```

Design rule:

- `run` is acceptable for single commands and short command lines
- once a step needs substantial shell logic, move it to a separate script and invoke that script

## Execution order

For a manifest-backed component, the default execution order is:

1. validate `[when]`
2. create `mkdirs`
3. install `packages`
4. apply `symlinks`
5. apply `copies`
6. execute `steps` in order

This keeps the model simple and predictable.

## `setup/default.toml`

`dfl setup` should read a repo-level manifest at `setup/default.toml`.

This file is similar to `install.toml`, but it exists for repo-wide setup orchestration rather than
for a single component.

It should support:

- top-level setup constraints via `[when]`
- `[[packages]]` using the same schema as component manifests
- `[[repos]]` for cloning and updating required repositories
- `[[steps]]` using the same schema as component manifests
- `[[components]]` entries for component selection and conditional inclusion

Suggested initial shape:

```toml
[repo_defaults]
transport = "inherit"

[[components]]
name = "fish"

[[components]]
name = "nvim"

[[components]]
name = "osx-tuning"
when_os = ["mac"]

[[packages]]
manager = "brew"
names = ["dff"]
tap = "elentok/stuff"

[[repos]]
name = "notes"
github = "elentok/notes"
path = "~/notes"

[[repos]]
name = "work-wiki"
github = "myorg/wiki"
path = "~/src/wiki"
transport = "https"

[[steps]]
name = "cache deno scripts"
run = "cd extra/scripts/deno && deno cache ./**/*.ts"
```

Setup-level `[[steps]]` are the place for repo-wide actions that are not naturally owned by a
single component, such as bootstrapping caches or creating top-level convenience symlinks.

`[[components]]` entries should support at least:

- `name`: required
- `when_os`: optional list of `mac`, `linux`, `wsl`
- `when_linux_distro`: optional list such as `debian`, `ubuntu`
- `when_features`: optional feature tags such as `gui` or `kde`

`[[repos]]` entries should support at least:

- `name`: required
- `path`: required
- either `github` or `url`: required
- `transport`: optional `inherit`, `ssh`, or `https`
- `when_os`: optional list of `mac`, `linux`, `wsl`
- `when_linux_distro`: optional list such as `debian`, `ubuntu`
- `when_features`: optional feature tags such as `gui` or `kde`

`[repo_defaults]` should support:

- `transport`: default transport policy for generated repo URLs; initial values should be
  `inherit`, `ssh`, or `https`

Repo URL resolution rules:

- if `url` is present on a repo entry, use it directly
- otherwise `github = "owner/name"` should be expanded into a clone URL
- `transport = "inherit"` means infer the URL style from the dotfiles repo `origin` remote; if the
  origin is not a clear GitHub SSH or GitHub HTTPS remote, fall back to HTTPS
- per-repo `transport` overrides the default when a repo needs different behavior

Generated GitHub URL forms:

- SSH: `git@github.com:owner/name.git`
- HTTPS: `https://github.com/owner/name.git`

Repo execution behavior:

- if the target path does not exist, clone the repo
- if the target path exists and is a Git checkout, run `git pull --ff-only`
- if the pull fails because the branch diverged, report the repo as `failed` with an explicit
  divergence message

`osx-tuning` should remain a normal component and be included from `setup/default.toml` using a
conditional `[[components]]` entry rather than being treated as a special built-in case.

## Dry-run behavior

`--dry-run` should be a first-class mode for both `dfl setup` and `dfl install`.

In dry-run mode:

- show the component resolution result
- show which repos would clone, update, skip, or fail precondition checks
- show which filesystem changes would happen
- show which package installs would run
- show which steps would execute or skip
- do not modify the machine

## Error handling

Requirements:

- each component is reported as `success`, `skipped`, or `failed`
- a failed step should fail the component
- `dfl install a b c` should continue or stop based on a top-level policy flag
- reporting should preserve step nesting where useful, so component summaries can show both
  high-level operations and the sub-operations beneath them

Suggested default:

- `dfl install a b c` stops on first failure
- future flag: `--keep-going`

## Migration strategy

### Phase 1

- Implement `dfl setup`
- Implement `dfl install`
- Implement core runtime commands
- Keep existing shell installers
- Keep `framework.sh` as a compatibility layer that can delegate to `dfl`
- Avoid a repo-wide script rewrite in the first milestone

### Phase 2

- Add `install.toml`
- Convert the simplest components first
- Keep support for shell installers for complex components

### Phase 3

- Move more package/bootstrap logic into Go
- shrink the remaining shell surface deliberately

## Migration examples

### `core/tmux`

Current behavior:

- symlink `tmux.conf`
- on macOS, install `tmux-256color` terminfo
- run plugin installer script

Recommended milestone 1 shell installer:

```bash
#!/usr/bin/env bash
set -euo pipefail

dfl symlink tmux.conf ~/.tmux.conf

if dfl os is-mac; then
  dfl step-start "Setting up tmux-256color terminfo"
  if infocmp tmux-256color >/dev/null 2>&1; then
    dfl step-end --skip "already set up"
  else
    if ./install-tmux-256color; then
      dfl step-end --success
    else
      dfl step-end --error
      exit 1
    fi
  fi
fi

dfl shell "Installing tmux plugins" -- "$DFL_COMPONENT_ROOT/install-plugins"
```

Later manifest candidate:

```toml
[symlinks]
"tmux.conf" = "~/.tmux.conf"

[[steps]]
name = "tmux-256color"
os = ["mac"]
if_not = "infocmp tmux-256color >/dev/null 2>&1"
run = "./install-tmux-256color"

[[steps]]
name = "plugins"
run = "./install-plugins"
```

### `extra/ssh`

Current behavior:

- symlink `rc`
- symlink `config`
- create `~/.ssh/machine.config` if missing

This is a strong early `install.toml` candidate:

```toml
[symlinks]
"rc" = "~/.ssh/rc"
"config" = "~/.ssh/config"

[[steps]]
name = "machine config"
if_not = "test -e ~/.ssh/machine.config"
run = "printf '# vim: syntax=sshconfig\n' > ~/.ssh/machine.config"
```

### `core/nvim`

Current behavior:

- symlink `core/nvim` to `~/.config/nvim`
- run two headless Neovim setup commands
- build `blink.cmp` fuzzy library if missing

This should remain a shell installer for now.

Reason:

- the final build step contains non-trivial procedural logic
- the component has multiple imperative steps with stateful checks

The shell installer should still use `dfl symlink` and `dfl shell` where it improves structure.

## Implementation decisions

- `dfl setup` should read `setup/default.toml` rather than using a hardcoded component list
- repo-wide setup concerns such as shared packages, repo synchronization, and non-component setup
  steps should live in `setup/default.toml`
- repo synchronization should be a first-class `[[repos]]` section in `setup/default.toml`, and
  `--skip-repos` should skip that phase
- repos should clone if missing and otherwise run `git pull --ff-only`
- GitHub repo transport should default to inheriting the dotfiles repo `origin` remote style, with
  per-repo overrides for `ssh` or `https`
- package definitions should move out of `packages.cfg` into TOML now, but remain repo-level before
  later migration into component `install.toml` files
- package groups should use `[[packages]]` entries with `names = [...]` and optional conditions
- setup-level `[[steps]]` should use the same model as component-manifest `[[steps]]`
- milestone 1 should keep `framework.sh` as a compatibility layer and migrate installers
  incrementally rather than rewriting every install script up front
- backups should use `<target>.backup` by default and fall back to a timestamped backup name when
  that path already exists
