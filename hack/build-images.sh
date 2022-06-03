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
## Uses Dockerfile.dev which will build all entrypoints in the same image.

if [ $# -lt 2 ]
then
  echo 'Usage:'
  echo '  ./build-images.sh IMAGE_NAME_PREFIX IMAGE_TAG'
  exit 1
fi

# Positional arguments.
IMAGE_NAME_PREFIX="$1"
if [[ -z "${IMAGE_NAME_PREFIX}" ]]
then
  echo 'Error: IMAGE_NAME_PREFIX cannot be empty'
  exit 2
fi

IMAGE_TAG="$2"
if [[ -z "${IMAGE_TAG}" ]]
then
  echo 'Error: IMAGE_TAG cannot be empty'
  exit 2
fi

# Build all images.
# Note that in the development build, all entrypoints are bundled in the same image.
while IFS= read -r IMAGE; do
  TARGET_IMAGE="${IMAGE_NAME_PREFIX}/${IMAGE}:${IMAGE_TAG}"
  docker build --file=Dockerfile.dev -t "${TARGET_IMAGE}" .
done < ./hack/docker-images.txt
