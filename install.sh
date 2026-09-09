#!/bin/sh

set -eu

repository="${PICKLE_REPOSITORY:-sriramsme/pickle}"
version="${PICKLE_VERSION:-latest}"
install_directory="$HOME/.local/bin"
install_path="$install_directory/pickle"

fail() {
  printf 'pickle: %s\n' "$1" >&2
  exit 1
}

download() {
  url="$1"
  output="$2"
  if command -v curl >/dev/null 2>&1; then
    curl --fail --location --retry 3 --silent --show-error --output "$output" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget --quiet --output-document="$output" "$url"
  else
    fail "curl or wget is required"
  fi
}

verify() {
  file="$1"
  name="${file##*/}"
  expected="$(awk -v name="$name" '$2 == name { print $1 }' "$temporary_directory/checksums.txt")"
  [ -n "$expected" ] || fail "checksum for $name was not found"

  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$file" | awk '{ print $1 }')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$file" | awk '{ print $1 }')"
  else
    fail "sha256sum or shasum is required"
  fi
  [ "$actual" = "$expected" ] || fail "checksum verification failed for $name"
}

[ "$(uname -s)" = "Linux" ] || fail "the Pickle host currently requires Linux"
case "$(uname -m)" in
  x86_64 | amd64) architecture="amd64" ;;
  aarch64 | arm64) architecture="arm64" ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

command -v tmux >/dev/null 2>&1 || fail "tmux is required; install it and run this script again"

case "$version" in
  *[!0-9A-Za-z._-]* | "") fail "invalid PICKLE_VERSION" ;;
esac

if [ "$version" = "latest" ]; then
  download_base="https://github.com/$repository/releases/latest/download"
else
  download_base="https://github.com/$repository/releases/download/$version"
fi

temporary_directory="$(mktemp -d)"
cleanup() {
  if [ -d "$temporary_directory" ]; then
    rm -r "$temporary_directory"
  fi
}
trap cleanup EXIT HUP INT TERM

asset="pickle_linux_$architecture"
download "$download_base/checksums.txt" "$temporary_directory/checksums.txt"
download "$download_base/$asset" "$temporary_directory/$asset"
verify "$temporary_directory/$asset"

mkdir -p "$install_directory"
cp "$temporary_directory/$asset" "$install_path.new"
chmod 755 "$install_path.new"
mv "$install_path.new" "$install_path"

service_started=false
if [ "${PICKLE_SKIP_SERVICE:-0}" != "1" ] && command -v systemctl >/dev/null 2>&1; then
  download "$download_base/pickle.service" "$temporary_directory/pickle.service"
  verify "$temporary_directory/pickle.service"
  service_directory="$HOME/.config/systemd/user"
  mkdir -p "$service_directory"
  cp "$temporary_directory/pickle.service" "$service_directory/pickle.service"

  if systemctl --user daemon-reload; then
    if systemctl --user is-active --quiet pickle; then
      systemctl --user restart pickle && service_started=true
    else
      systemctl --user enable --now pickle && service_started=true
    fi
  fi
fi

printf '\nPickle is installed at %s.\n' "$install_path"
if [ "$service_started" = true ]; then
  printf 'Open http://127.0.0.1:8080/setup\n'
  printf 'On a headless host, enable startup before login with:\n'
  printf '  sudo loginctl enable-linger %s\n' "$(id -un)"
else
  printf 'Start it with: %s\n' "$install_path"
  printf 'Then open http://127.0.0.1:8080/setup\n'
fi

if command -v tailscale >/dev/null 2>&1; then
  printf '\nFor private access from your other devices, run:\n'
  printf '  tailscale serve --bg 8080\n'
else
  printf '\nFor private remote access, install Tailscale:\n'
  printf '  https://tailscale.com/docs/install/linux\n'
fi
