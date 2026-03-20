#!/usr/bin/env bash
set -eo pipefail

# Ensure protoc plugins are on PATH.
export PATH="$PATH:$(go env GOPATH)/bin"

echo "Generating protobuf code..."
buf generate

# Move generated files from github.com/oasyce/chain/x/ to x/
for mod in settlement capability reputation datarights; do
  src="github.com/oasyce/chain/x/$mod/types"
  dst="x/$mod/types"
  if [ -d "$src" ]; then
    cp "$src"/*.pb.go "$dst/" 2>/dev/null || true
    cp "$src"/*.pb.gw.go "$dst/" 2>/dev/null || true
  fi
done

# Cleanup temporary directory
rm -rf github.com

echo "Protobuf generation complete."
