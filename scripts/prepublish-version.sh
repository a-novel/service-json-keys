#!/bin/bash

# Updates version references in a file using sed, replacing occurrences of <prefix>vX.Y.Z with
# <prefix>v<current-package-version>. Works on both GNU (Linux) and BSD (macOS) sed.
# Usage: ./prepublish-version.sh <prefix> <file>

set -e

if [ $# -ne 2 ]; then
    printf "Usage: %s <prefix> <file>\n" "$0" >&2
    exit 1
fi

VERSION="$(node -p "require('./package.json').version")"

case "$OSTYPE" in
    darwin*|bsd*)
        sed -i '' -E "s|($1)v[0-9.]+|\1v${VERSION}|g" "$2"
        ;;
    *)
        sed -i -E "s|($1)v[0-9.]+|\1v${VERSION}|g" "$2"
        ;;
esac
