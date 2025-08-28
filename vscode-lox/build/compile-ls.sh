#!/usr/bin/env bash

NAME="loxls"
WINDOWS_NAME="$NAME.exe"

rm -f "out/$NAME"
rm -f "out/$WINDOWS_NAME"

if [[ ! -v NO_LOXLS ]]; then
  name="$NAME"
  if [[ "$GOOS" == "windows" ]]; then
    name="$WINDOWS_NAME"
  fi
  go build -o "out/$name" ../loxls
fi
