#!/bin/zsh

set -e

zsh /Users/will/Projects/Go/kabao/backend/scripts/stop-go-node.sh
cd /Users/will/Projects/Go/kabao/backend
dlv debug . --headless --listen=127.0.0.1:2345 --api-version=2 --accept-multiclient --continue --output /tmp/__debug_bin

exit 0
