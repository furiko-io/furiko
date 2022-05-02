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

## Builds all Docker images to a specified image tag, but does not push them.
## Uses GoReleaser to build artifacts.

if [ $# -ne 1 ]
then
  echo 'Usage:'
  echo '  ./build-images.sh IMAGE_TAG'
  echo
  echo 'Optional environment variables:'
  echo '  GORELEASER: Path to goreleaser executable. Default: ./bin/goreleaser'
  exit 1
fi

# Positional arguments.
IMAGE_TAG="$1"

# Optional environment variables.
GORELEASER="${GORELEASER:-$(pwd)/bin/goreleaser}"

# Pass the IMAGE_TAG environment variable to GoReleaser.
export IMAGE_TAG

# Run GoReleaser to build all images in parallel. Pass IMAGE_TAG to specify the target image tag.
# TODO(irvinlim): This will also build all other artifacts, which is not needed here.
"${GORELEASER}" release --snapshot --rm-dist
