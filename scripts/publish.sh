#!/bin/bash

# Bumps the package version to the value passed as $1, commits the change, tags it, and pushes to the remote.
# Usage: ./publish.sh <new-version>

set -e

if [ $# -ne 1 ]; then
    printf "Usage: %s <new-version>\n" "$0" >&2
    exit 1
fi

pnpm version "$1" --workspaces --workspaces-update=false --no-git-tag-version
pnpm prepublish:doc

VERSION="$(node -p "require('./package.json').version")"

git add -A
git commit -m "${VERSION}"
git tag "v${VERSION}"
git push
git push --tags
