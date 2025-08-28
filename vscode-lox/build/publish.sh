#!/usr/bin/env bash

GOOS=windows GOARCH=amd64 npx @vscode/vsce publish --target win32-x64
GOOS=windows GOARCH=arm64 npx @vscode/vsce publish --target win32-arm64
GOOS=linux GOARCH=amd64 npx @vscode/vsce publish --target linux-x64
GOOS=linux GOARCH=arm64 npx @vscode/vsce publish --target linux-arm64
GOOS=linux GOARCH=arm npx @vscode/vsce publish --target linux-armhf
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 npx @vscode/vsce publish --target alpine-x64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 npx @vscode/vsce publish --target alpine-arm64
GOOS=darwin GOARCH=amd64 npx @vscode/vsce publish --target darwin-x64
GOOS=darwin GOARCH=arm64 npx @vscode/vsce publish --target darwin-arm64
NO_LOXLS=1 npx @vscode/vsce publish
