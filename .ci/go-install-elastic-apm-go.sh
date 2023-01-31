#!/usr/bin/env bash
## This script updates the version for the APM agent that's provided.

set -eo pipefail

go get "go.elastic.co/apm/...@$1"
go mod tidy
