#!/bin/sh
set -eu

echo "Installing dfl..."

bin="$HOME/.local/bin"
target="$bin/dfl"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

os="$(uname -s)"
case "$(uname -m)" in
x86_64 | amd64) arch=x86_64 ;;
arm64 | aarch64) arch=arm64 ;;
*)
  echo "unsupported architecture: $(uname -m)" >&2
  exit 1
  ;;
esac

asset="dfl_${os}_${arch}.tar.gz"
url="https://github.com/elentok/dfl/releases/latest/download/$asset"
archive="$tmp/$asset"

download() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$archive"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$archive" "$url"
  else
    echo "curl or wget is required" >&2
    exit 1
  fi
}

mkdir -p "$bin"
if [ ! -x "$target" ]; then
  download || {
    echo "failed to download $url" >&2
    exit 1
  }
  tar -xzf "$archive" -C "$tmp" dfl
  install -m 755 "$tmp/dfl" "$target"
fi

export PATH="$bin:$PATH"
exec "$target" self install "$@"
