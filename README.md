# dfl

`dfl` is a Go-based runtime for bootstrapping and maintaining a dotfiles repo.

It is designed to replace an ad hoc shell framework with a single binary that can:

- run repo setup from a stable entrypoint
- install named dotfiles components
- provide reusable runtime commands for installer scripts
- bootstrap from a minimal shell installer
- self-update from GitHub releases

## Install

To set up `dfl` for your dotfiles repo, start from the dotfiles repo bootstrap script.

Example using [`elentok/dotfiles/bootstrap`](https://github.com/elentok/dotfiles/blob/main/bootstrap):

```sh
mkdir -p ~/.dotfiles
cd ~/.dotfiles
curl -fsSL https://raw.githubusercontent.com/elentok/dotfiles/main/bootstrap -o bootstrap
chmod +x bootstrap
./bootstrap
```

That dotfiles-side bootstrap flow:

1. resolves the repo root from the script location
2. installs `dfl` to `~/.local/bin/dfl` if it is missing
3. runs `dfl setup --repo <repo-root>`

This repository also ships the public installer shim used by bootstrap:

```sh
./install-dfl.sh
```

That command is for installing the `dfl` binary itself. After `dfl` is installed, the normal maintenance entrypoint is:

```sh
dfl update
```

## Core Commands

Run the repo setup script:

```sh
dfl setup
```

Target a specific repo explicitly:

```sh
dfl setup --repo ~/.dotfiles
```

Install one or more components from the resolved repo:

```sh
dfl install fish
dfl install fish nvim git
dfl i tmux
```

Preview the update flow without making changes:

```sh
dfl --dry-run update --repo ~/.dotfiles
```

Show the resolved repo root:

```sh
dfl repo-root
```

Print the binary version:

```sh
dfl version
```

## Runtime Commands

`dfl` also exposes lower-level runtime commands intended for use by component installer scripts.

Examples:

```sh
dfl has-command git
dfl os is-mac
dfl pkg brew install ripgrep
dfl symlink tmux.conf ~/.tmux.conf
dfl copy config.toml ~/.config/myapp/config.toml
dfl mkdir ~/.config/myapp
dfl backup ~/.gitconfig
dfl shell "Reload config" -- sh -c 'echo hello'
```

These commands are designed to produce consistent step-style output and to support `--dry-run`.

## Release Artifacts

Releases are published for:

- macOS `amd64`
- macOS `arm64`
- Linux `amd64`
- Linux `arm64`

Release archives are named like:

```text
dfl_Darwin_arm64.tar.gz
dfl_Linux_x86_64.tar.gz
```

## Development

Run the test suite:

```sh
make test
```

Build locally:

```sh
go build ./...
```

The release pipeline injects the version string with GoReleaser using `-X dfl/internal/buildinfo.Version=<tag>`.

## Notes

- `dfl setup` expects a dotfiles repo layout with a repo-specific `core/setup` entrypoint.
- `dfl install` resolves components from the target repo and executes their `install` scripts.
- The installer and updater converge on `~/.local/bin/dfl`.
