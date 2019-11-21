#!/usr/bin/env bash
set -euxo pipefail

if [ $# -lt 1 ]; then
  echo "usage: ${0} folder"
  exit 1
fi
AGENT_VERSION="${1}"

# Install Go using the same travis approach
echo "Installing ${GO_VERSION} with gimme."
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"

# Update agent dependencies
go get "go.elastic.co/apm/...@${AGENT_VERSION}"
go mod tidy

# Commit changes
#git add go.mod go.sum
#git commit -m "Bump version ${AGENT_VERSION}"
