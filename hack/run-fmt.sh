#!/bin/bash
# shellcheck disable=SC2046

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

set -euo pipefail

## Simple script that performs code formatting.

if [ $# -ne 1 ]
then
  echo 'Usage:'
  echo '  ./run-fmt.sh GO_PACKAGE_NAME'
  echo
  echo 'Optional environment variables:'
  echo '  GOIMPORTS: Path to goimports executable. Default: ./bin/goimports'
  exit 1
fi

# Positional arguments
GO_PACKAGE_NAME="$1"
if [[ -z "${GO_PACKAGE_NAME}" ]]
then
  echo 'Error: GO_PACKAGE_NAME cannot be empty'
  exit 2
fi

# Optional environment variables
GOIMPORTS="${GOIMPORTS:-$(pwd)/bin/goimports}"

# Run go fmt.
go fmt ./cmd/... $(go list -f "{{.Dir}}" ./pkg/... | grep -v pkg/generated)

# Run goimports to sort imports.
# Do not sort generated files.
./bin/goimports -w -local "${GO_PACKAGE_NAME}" ./cmd $(go list -f "{{.Dir}}" ./pkg/... | grep -v pkg/generated)
