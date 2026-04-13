#!/bin/bash

# Bumps the package version to the value passed as $1, commits the change, tags it, and pushes to the remote.
# Usage: ./publish.sh <new-version>

set -e

pnpm version $1 --workspaces --workspaces-update=false --no-git-tag-version
pnpm prepublish:doc

git add -A
git commit -m "$(node -p "require('./package.json').version")"
git tag v$(node -p "require('./package.json').version")
git push
git push --tags
