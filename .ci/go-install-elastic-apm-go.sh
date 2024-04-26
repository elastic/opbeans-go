#!/usr/bin/env bash
## This script updates the version for the APM agent that's provided.

set -eo pipefail

go get "go.elastic.co/apm/...@$1"

# git diff is a safe choice because it will be non-empty when changes need to be committed and it's also good for
# debugging.
git diff

go mod tidy

git diff