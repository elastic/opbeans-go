#!/usr/bin/env bash
## This script compares whether the given version defined in mod.go and the provided version
## are equally or not.

set -eo pipefail
test "$(go list -f '{{.Version}}' -m go.elastic.co/apm/v2)" != "$1"
exit $?
