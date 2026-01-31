#!/usr/bin/env bash

common_flags="--baseImagesUrl https://raw.githubusercontent.com/marcuscaisey/lox/master/vscode-lox/"
# shellcheck disable=SC2086
GOOS=windows GOARCH=amd64 npx @vscode/vsce publish --target win32-x64 $common_flags
# shellcheck disable=SC2086
GOOS=windows GOARCH=arm64 npx @vscode/vsce publish --target win32-arm64 $common_flags
# shellcheck disable=SC2086
GOOS=linux GOARCH=amd64 npx @vscode/vsce publish --target linux-x64 $common_flags
# shellcheck disable=SC2086
GOOS=linux GOARCH=arm64 npx @vscode/vsce publish --target linux-arm64 $common_flags
# shellcheck disable=SC2086
GOOS=linux GOARCH=arm npx @vscode/vsce publish --target linux-armhf $common_flags
# shellcheck disable=SC2086
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 npx @vscode/vsce publish --target alpine-x64 $common_flags
# shellcheck disable=SC2086
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 npx @vscode/vsce publish --target alpine-arm64 $common_flags
# shellcheck disable=SC2086
GOOS=darwin GOARCH=amd64 npx @vscode/vsce publish --target darwin-x64 $common_flags
# shellcheck disable=SC2086
GOOS=darwin GOARCH=arm64 npx @vscode/vsce publish --target darwin-arm64 $common_flags
# shellcheck disable=SC2086
NO_LOXLS=1 npx @vscode/vsce publish $common_flags
