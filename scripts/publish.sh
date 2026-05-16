#!/bin/bash

# Bumps the package version to the value passed as $1, commits the change, tags it, and pushes to the remote.
# Usage: ./publish.sh <new-version>

set -e

if [ $# -ne 1 ]; then
    printf "Usage: %s <new-version>\n" "$0" >&2
    exit 1
fi

# npm (not pnpm) owns the version bump: --workspaces / --workspaces-update are
# npm flags. Older pnpm proxied `pnpm version` to npm so they were accepted;
# pnpm >=11 parses `version` itself and rejects them ("Unknown options:
# 'workspaces', 'workspaces-update'"). The root package.json declares an npm
# `workspaces` array (incl. "."), so `npm version --workspaces` bumps the root
# and every member; --workspaces-update=false keeps interdependency ranges and
# --no-git-tag-version leaves the commit/tag to this script.
npm version "$1" --workspaces --workspaces-update=false --no-git-tag-version
pnpm prepublish:doc

VERSION="$(node -p "require('./package.json').version")"

git add -A
git commit -m "${VERSION}"
git tag "v${VERSION}"
git push
git push --tags
