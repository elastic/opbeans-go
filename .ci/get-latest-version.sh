#!/usr/bin/env bash
## This script prints the Version for go.elastic.co/apm/v2 in the stdout

set -eo pipefail
go list -json -u -m go.elastic.co/apm/v2 | jq -r .Version
exit $?
