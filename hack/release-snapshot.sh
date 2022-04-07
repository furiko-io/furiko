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

## Simple script that builds a snapshot of the current repository and pushes an image to Docker Hub.
## Uses GoReleaser to build artifacts.
## Docker image will be released as using the `<NEXT TAG>-next` image tag.

# Build snapshot into dist.
make -s goreleaser
./bin/goreleaser release --snapshot --rm-dist

# Get version number (excludes the v prefix).
VERSION=$(jq .version dist/metadata.json --raw-output)

# Tag and push all images.
REPOS=(
  'furikoio/execution-controller'
  'furikoio/execution-webhook'
)
for REPO in "${REPOS[@]}"
do
  docker push "${REPO}:v${VERSION}"
done
