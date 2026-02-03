#!/usr/bin/env bash
set -euo pipefail

# Generate Go code from proto definitions.
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc
#
# Install:
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$ROOT_DIR/api/proto"
OUT_DIR="$PROTO_DIR/bff"

mkdir -p "$OUT_DIR"

protoc \
  --proto_path="$PROTO_DIR" \
  --go_out="$ROOT_DIR" \
  --go_opt=paths=source_relative \
  --go-grpc_out="$ROOT_DIR" \
  --go-grpc_opt=paths=source_relative \
  "$PROTO_DIR/bff.proto"

echo "Proto generation complete: $OUT_DIR"
