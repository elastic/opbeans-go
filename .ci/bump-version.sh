#!/usr/bin/env bash
set -exo pipefail

if [ $# -lt 1 ]; then
  echo "usage: ${0} Go_Agent_Version"
  exit 1
fi
AGENT_VERSION="${1}"

# Prepare the Go environment
if [ -z ${GO_VERSION+x} ] ; then
  echo "Using the already installed golang version."
else
  # Install Go using the same travis approach
  echo "Installing ${GO_VERSION} with gimme."
  eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${GO_VERSION} bash)"
fi

# Update agent dependencies
go get "go.elastic.co/apm/...@${AGENT_VERSION}"
go mod tidy

## Bump agent version in the Dockerfile
sed -ibck "s#\(org.label-schema.version=\)\(\".*\"\)\(.*\)#\1\"${AGENT_VERSION}\"\3#g" Dockerfile

# Commit changes
git add go.mod go.sum Dockerfile
git commit -m "Bump version ${AGENT_VERSION}"
