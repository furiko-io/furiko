#!/usr/bin/env bash

set -euxo pipefail

# Run go test for each Go submodule and combine coverage.
# Exclude the .cache directory.
# Cannot use -execdir because find always exits with 0 even if `go test` returns non-zero exit code.
find "$(pwd)" -not -path '*/\.*' -name go.mod -printf "%h\n" |\
  xargs -I {} bash -c "cd {} && go test -coverpkg=./... -coverprofile ./coverage.cov ./..."

# Combine all coverage files, skipping first line of each file
echo "mode: set" > combined.cov
find . -name coverage.cov -exec sh -c 'tail -n +2 $1' shell {} \; >> combined.cov
go tool cover -func=combined.cov
