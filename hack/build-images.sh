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

DOCKERFILE="Dockerfile.dev"
IMAGE_NAME_PREFIX=""
IMAGE_TAG=""

while getopts ":f:p:t:" opt; do
  case $opt in
    f)
      # Defines the Dockerfile to use.
      # Example: Dockerfile.dev
      DOCKERFILE="$OPTARG"
      ;;
    p)
      # Defines the image name prefix of the built image.
      # Example: ghcr.io/furiko-io
      IMAGE_NAME_PREFIX="$OPTARG"
      ;;
    t)
      # Defines the image tag of the build image.
      # Example: latest
      IMAGE_TAG="$OPTARG"
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      exit 2
      ;;
    :)
      echo "Usage: ./build-images.sh -p IMAGE_NAME_PREFIX -t IMAGE_TAG [-f DOCKERFILE]" >&2
      exit 2
      ;;
  esac
done

if [[ -z "${DOCKERFILE}" || -z "${IMAGE_NAME_PREFIX}" || -z "${IMAGE_TAG}" ]]
then
  echo "Usage: ./build-images.sh -p IMAGE_NAME_PREFIX -t IMAGE_TAG [-f DOCKERFILE]" >&2
  exit 2
fi

# Build all images.
while IFS= read -r IMAGE; do
  IMAGE_NAME="${IMAGE_NAME_PREFIX}/${IMAGE}:${IMAGE_TAG}"

  # Note that $IMAGE has the same name as the target stage in the Docker multi-stage build.
  docker build --file="${DOCKERFILE}" --target="${IMAGE}" -t "${IMAGE_NAME}" .
done < ./hack/docker-images.txt
