#!/bin/bash

# Bumps the package version to the value passed as $1, commits the change, tags it, and pushes to the remote.
# Usage: ./publish.sh <new-version>

set -e

if [ $# -ne 1 ]; then
    printf "Usage: %s <new-version>\n" "$0" >&2
    exit 1
fi

# Bump the root project and every workspace member. --workspaces /
# --workspaces-update are npm flags; pnpm <11 silently proxied `pnpm version`
# to npm so they worked, but pnpm >=11 has a native `version` command that
# rejects them ("Unknown options: 'workspaces', 'workspaces-update'"). The
# pnpm-native equivalent is --recursive, which versions the workspace root and
# every member in one pass; pnpm always skips the git commit/tag in recursive
# mode, so this script still owns the commit/tag/push below.
pnpm version "$1" --recursive
pnpm prepublish:doc

VERSION="$(node -p "require('./package.json').version")"

git add -A
git commit -m "${VERSION}"
git tag "v${VERSION}"
git push
git push --tags
