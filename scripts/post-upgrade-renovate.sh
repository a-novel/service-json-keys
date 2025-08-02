#!/bin/bash

set -e

export NVM_DIR="$HOME/.nvm" # Prevent nvm from using the repo as root.
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
\. "$HOME/.nvm/nvm.sh"
nvm install node

npx -y prettier . --write
