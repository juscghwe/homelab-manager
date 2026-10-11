#!/usr/bin/env bash
# Run the same read-only validation locally and in GitHub Actions.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
    printf 'Go formatting differs; run gofmt on:\n%s\n' "$unformatted" >&2
    exit 1
fi

go vet ./...
go test ./...
CGO_ENABLED=1 go test -race ./...
# Agent libraries must also build without a C compiler/runtime dependency.
CGO_ENABLED=0 go build ./...
