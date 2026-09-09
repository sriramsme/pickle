#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$repo_root"
pnpm --dir web build
mkdir -p build

cp contrib/systemd/pickle.service build/pickle.service
artifacts=("pickle.service")
for arch in amd64 arm64; do
  artifact="pickle_linux_${arch}"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" \
    go build -trimpath -ldflags="-s -w" -o "build/$artifact" ./cmd/server
  artifacts+=("$artifact")
done

(
  cd build
  sha256sum "${artifacts[@]}" > checksums.txt
)
