#!/bin/bash
set -euo pipefail

golox_dir="$(dirname "$0")"
build_path="$golox_dir/../build/golox"
pushd "$golox_dir" >/dev/null
go build -o "$build_path" .
popd >/dev/null
"$build_path" "$@"
