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
## Docker images will be pushed to Docker Hub using a snapshot version tag (e.g. `v1.2.3-next`).

# Optionally specify IMAGE_TAG, otherwise will default to `(current tag with patch version incremented)-next`.
# Environment variable is expected to be propagated to GoReleaser.
export IMAGE_TAG

# Build snapshot into dist.
curl -sL https://git.io/goreleaser | bash -s -- release --snapshot --rm-dist

# Get version number (excludes the v prefix).
VERSION=$(jq .version dist/metadata.json --raw-output)

# Tag and push all images.
REPOS=(
  'furikoio/execution-controller'
  'furikoio/execution-webhook'
)
for REPO in "${REPOS[@]}"
do
  VERSION_TAG="${IMAGE_TAG:-v${VERSION}}"

  # The versioned image tag is like v1.2.3-next, or will default to the IMAGE_TAG environment variable specified.
  VERSIONED_IMAGE="${REPO}:${VERSION_TAG}"

  # The snapshot image tag always points to the latest snapshot.
  SNAPSHOT_IMAGE="${REPO}:snapshot"

  docker tag "${VERSIONED_IMAGE}" "${SNAPSHOT_IMAGE}"
  docker push "${VERSIONED_IMAGE}"
  docker push "${SNAPSHOT_IMAGE}"
done
