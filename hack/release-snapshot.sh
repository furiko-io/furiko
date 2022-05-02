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

## Simple script that builds a snapshot of the current repository as Docker images and pushes them to Docker Hub.
## Docker images will be pushed to Docker Hub using a snapshot version tag (e.g. `v1.2.3-next`).

# Optional environment variables.
GORELEASER="${GORELEASER:-$(pwd)/bin/goreleaser}"
IMAGE_NAME_PREFIX="${IMAGE_NAME_PREFIX:-"docker.io/furikoio"}"

# Optionally specify IMAGE_TAG, otherwise will default to `(current tag with patch version incremented)-next`.
# Environment variable is expected to be propagated to GoReleaser.
export IMAGE_TAG

# Run GoReleaser to build all images in parallel. Builds all artifacts into dist.
# TODO(irvinlim): This will also build all other artifacts, which is not needed here.
"${GORELEASER}" release --snapshot --rm-dist

# If IMAGE_TAG is not specified, we need to read the built image tag from the metadata file (excludes the v prefix).
VERSION=$(jq .version dist/metadata.json --raw-output)
VERSION_TAG="${IMAGE_TAG:-v${VERSION}}"

# Push images with the versioned tag (e.g. v1.2.3-next).
./hack/push-images.sh "${IMAGE_NAME_PREFIX}" "${VERSION_TAG}"

# Push images with the "snapshot" tag.
./hack/push-images.sh "${IMAGE_NAME_PREFIX}" "${VERSION_TAG}" "${IMAGE_NAME_PREFIX}" "snapshot"
