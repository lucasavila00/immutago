#!/bin/sh
set -eu

# Library entry points are exercised by tests, not necessarily by the CLI.
findings=$(go tool deadcode -test ./...)
if [ -n "$findings" ]; then
    printf '%s\n' "$findings"
    exit 1
fi
