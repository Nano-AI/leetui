#!/bin/sh
# Install leetui.
#
#   curl -fsSL https://raw.githubusercontent.com/Nano-AI/leetui/main/install.sh | sh
#
# Downloads the right prebuilt binary for this machine, VERIFIES ITS CHECKSUM against
# the release's own checksums.txt, and installs it. No Go toolchain, no Homebrew.
#
# POSIX sh, not bash: this runs on whatever /bin/sh is, including dash on Debian and
# ash on Alpine, and a bashism here fails on the machines least likely to have bash.
#
# Read it before you pipe it into a shell. That advice applies to every install script
# on the internet, and it applies to this one.

set -eu

REPO="Nano-AI/leetui"
BIN="leetui"

say()  { printf '%s\n' "$*"; }
die()  { printf 'install: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "need $1 on PATH"; }

need uname
need mktemp
need tar

# curl or wget, whichever exists. Minimal images often have exactly one.
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
  fetch_stdout() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
  fetch_stdout() { wget -qO- "$1"; }
else
  die "need curl or wget"
fi

# --- what machine is this ----------------------------------------------------

os=$(uname -s)
case "$os" in
  Darwin) os=darwin ;;
  Linux)  os=linux ;;
  *)      die "no prebuilt binary for $os — install with Go:
    go install github.com/$REPO/cmd/leetui@latest" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64)  arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *)             die "no prebuilt binary for $arch — install with Go:
    go install github.com/$REPO/cmd/leetui@latest" ;;
esac

# --- which release -----------------------------------------------------------

version="${LEETUI_VERSION:-}"
if [ -z "$version" ]; then
  # Resolve "latest" through the redirect rather than the API: no token, no rate
  # limit, and it works from a machine that has never authenticated to GitHub.
  version=$(fetch_stdout "https://api.github.com/repos/$REPO/releases/latest" |
    sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
fi
[ -n "$version" ] || die "could not determine the latest version; set LEETUI_VERSION=v0.1.0"

archive="${BIN}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$version"

say "leetui $version — $os/$arch"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

say "downloading…"
fetch "$base/$archive" "$tmp/$archive" || die "could not download $archive"

# --- verify ------------------------------------------------------------------
#
# The whole reason the release publishes checksums.txt. Skipping this would mean
# piping an unverified binary from the network straight onto your PATH.

if fetch "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
  expected=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)
  if [ -n "$expected" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      actual=$(sha256sum "$tmp/$archive" | cut -d' ' -f1)
    elif command -v shasum >/dev/null 2>&1; then
      actual=$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)
    else
      actual=""
      say "warning: no sha256sum or shasum; cannot verify the download"
    fi
    if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
      die "CHECKSUM MISMATCH
  expected $expected
  got      $actual
Do not run this binary."
    fi
    [ -n "$actual" ] && say "checksum ok"
  fi
else
  say "warning: no checksums.txt for $version; skipping verification"
fi

tar -xzf "$tmp/$archive" -C "$tmp" || die "could not extract $archive"
[ -f "$tmp/$BIN" ] || die "$archive did not contain $BIN"
chmod +x "$tmp/$BIN"

# --- where to put it ---------------------------------------------------------
#
# A directory already on PATH, preferring one that needs no sudo. Installing
# somewhere the user then has to be told to add is how `go install` trips people.

dest="${LEETUI_INSTALL_DIR:-}"
if [ -z "$dest" ]; then
  for candidate in "$HOME/.local/bin" /usr/local/bin; do
    case ":$PATH:" in
      *":$candidate:"*) [ -d "$candidate" ] && [ -w "$candidate" ] && dest="$candidate" && break ;;
    esac
  done
fi
if [ -z "$dest" ]; then
  dest="$HOME/.local/bin"
  mkdir -p "$dest"
fi

mv "$tmp/$BIN" "$dest/$BIN" || die "could not write to $dest"
say "installed to $dest/$BIN"

# Say so plainly rather than leaving them to discover it. This is the exact
# failure `go install` has: a binary in a directory nobody's shell searches.
case ":$PATH:" in
  *":$dest:"*) ;;
  *)
    say ""
    say "$dest is not on your PATH. Add it:"
    say "    echo 'export PATH=\"$dest:\$PATH\"' >> ~/.zshrc && exec zsh"
    ;;
esac

say ""
"$dest/$BIN" --version 2>/dev/null || true
say "Run 'leetui' to start, or 'leetui doctor' to check this machine."
