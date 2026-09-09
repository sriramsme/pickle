#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="${1:-dev}"

if [[ ! "$version" =~ ^[0-9A-Za-z._-]+$ ]]; then
  echo "version may contain only letters, numbers, dots, underscores, and hyphens" >&2
  exit 1
fi

cd "$repo_root"
pnpm --dir web build
mkdir -p build

artifacts=()
for arch in amd64 arm64; do
  artifact="pickle_${version}_linux_${arch}"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" \
    go build -trimpath -ldflags="-s -w" -o "build/$artifact" ./cmd/server
  artifacts+=("$artifact")
done

(
  cd build
  sha256sum "${artifacts[@]}" > checksums.txt
)
