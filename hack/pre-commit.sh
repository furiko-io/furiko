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

set -euo pipefail

## Sample pre-commit Git hook that will format, lint and test changes.
## Reduce back-and-forth from fixing CI errors by running the CI locally.
##
## To enable this, run the following:
##   ln -s `pwd`/hack/pre-commit.sh .git/hooks/pre-commit

# Format the repo.
make fmt

# Run preflight checks.
make lint
make test
