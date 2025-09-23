#!/bin/bash
set -euo pipefail

jloxify_dir="$(realpath "$(dirname "$0")")"
build_path="$jloxify_dir/../../../build/jloxify"
pushd "$jloxify_dir" >/dev/null
go build -o "$build_path" .
popd >/dev/null
"$build_path" "$@"
