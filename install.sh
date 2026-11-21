#!/bin/sh
# Install the latest link-buzz-agent release binary, after checking its sha256
# against the release's checksums.txt.
#   curl -fsSL https://raw.githubusercontent.com/untoldecay/link-buzz-agent/main/install.sh | sh
# INSTALL_DIR overrides the target directory; VERSION pins a release (e.g. v0.2.0).
set -eu

repo=untoldecay/link-buzz-agent
case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  arm64 | aarch64) arch=arm64 ;;
  x86_64 | amd64) arch=amd64 ;;
  *) echo "unsupported CPU: $(uname -m)" >&2; exit 1 ;;
esac

version=${VERSION:-$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" |
  sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)}
[ -n "$version" ] || { echo "could not find the latest release" >&2; exit 1; }

bin=link-buzz-agent-$os-$arch
url=https://github.com/$repo/releases/download/$version
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL -o "$tmp/$bin" "$url/$bin"
curl -fsSL -o "$tmp/checksums.txt" "$url/checksums.txt"

want=$(awk -v f="$bin" '$2 == f {print $1}' "$tmp/checksums.txt")
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$tmp/$bin" | awk '{print $1}')
else
  got=$(shasum -a 256 "$tmp/$bin" | awk '{print $1}')
fi
[ -n "$want" ] && [ "$want" = "$got" ] || { echo "checksum mismatch for $bin: refusing to install" >&2; exit 1; }

if [ -z "${INSTALL_DIR:-}" ]; then
  if [ -w /usr/local/bin ]; then INSTALL_DIR=/usr/local/bin; else INSTALL_DIR=$HOME/.local/bin; fi
fi
mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/$bin" "$INSTALL_DIR/link-buzz-agent"
echo "installed link-buzz-agent $version to $INSTALL_DIR (sha256 $got)"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "add it to your PATH:  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc && exec zsh" ;;
esac
