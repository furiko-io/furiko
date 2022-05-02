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

## Tags and pushes Docker images that have already been built.

if [ $# -lt 2 ]
then
  echo 'Usage:'
  echo '  ./push-images.sh IMAGE_NAME_PREFIX IMAGE_TAG [NEW_IMAGE_NAME_PREFIX] [NEW_IMAGE_TAG]'
  exit 1
fi

# Positional arguments
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

NEW_IMAGE_NAME_PREFIX="${3:-"${IMAGE_NAME_PREFIX}"}"
NEW_IMAGE_TAG="${4:-"${IMAGE_TAG}"}"

# Tag and push all images.
while IFS= read -r IMAGE; do
  SRC_IMAGE="${IMAGE_NAME_PREFIX}/${IMAGE}:${IMAGE_TAG}"
  NEW_IMAGE="${NEW_IMAGE_NAME_PREFIX}/${IMAGE}:${NEW_IMAGE_TAG}"

  # Default the target image to be the source image.
  TARGET_IMAGE="${SRC_IMAGE}"

  # Re-tag the image and push that tag instead.
  if [ "${SRC_IMAGE}" != "${NEW_IMAGE}" ]; then
    TARGET_IMAGE="${NEW_IMAGE}"
    docker tag "${SRC_IMAGE}" "${TARGET_IMAGE}"
  fi

  # Push the target image.
  docker push "${TARGET_IMAGE}"
done < ./hack/docker-images.txt
