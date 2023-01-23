#!/usr/bin/env bash

#
# Copyright 2022 The Furiko Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set -euxo pipefail

# Run go test for each Go submodule and combine coverage.
# Exclude the .cache directory.
# Cannot use -execdir because find always exits with 0 even if `go test` returns non-zero exit code.
find "$(pwd)" -not -path '*/\.*' -name go.mod -print0 |\
  xargs -I {} bash -c "cd $(dirname {}) && go test -coverpkg=./... -coverprofile ./coverage.cov ./..."

# Combine all coverage files, skipping first line of each file
echo "mode: set" > combined.cov
find . -name coverage.cov -exec sh -c 'tail -n +2 $1' shell {} \; >> combined.cov
go tool cover -func=combined.cov
