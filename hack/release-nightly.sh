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

## Simple script that performs a nightly release to build and push container images.

IMAGE_NAME_PREFIX="ghcr.io/furiko-io"
IMAGE_TAG="nightly"

while getopts ":p:t:" opt; do
  case $opt in
    p)
      # Defines the image name prefix of the built image.
      # Example: ghcr.io/furiko-io
      IMAGE_NAME_PREFIX="$OPTARG"
      ;;
    t)
      # Defines the image tag of the build image.
      # Example: nightly
      IMAGE_TAG="$OPTARG"
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      exit 2
      ;;
    :)
      echo "Usage: ./release-nightly.sh -p IMAGE_NAME_PREFIX -t IMAGE_TAG" >&2
      exit 2
      ;;
  esac
done

if [[ -z "${IMAGE_NAME_PREFIX}" || -z "${IMAGE_TAG}" ]]
then
  echo "Usage: ./release-nightly.sh -p IMAGE_NAME_PREFIX -t IMAGE_TAG" >&2
  exit 2
fi

# Build and push all images using the dev Dockerfile.
./hack/build-images.sh -f "Dockerfile.dev" -p "${IMAGE_NAME_PREFIX}" -t "${IMAGE_TAG}"
./hack/push-images.sh -p "${IMAGE_NAME_PREFIX}" -t "${IMAGE_TAG}"
