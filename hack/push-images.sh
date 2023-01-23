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

IMAGE_NAME_PREFIX=""
IMAGE_TAG=""
NEW_IMAGE_NAME_PREFIX=""
NEW_IMAGE_TAG=""

while getopts ":p:t:P:T:" opt; do
  case $opt in
    p)
      # Defines the image name prefix to push.
      # Example: ghcr.io/furiko-io
      IMAGE_NAME_PREFIX="$OPTARG"
      ;;
    t)
      # Defines the image tag to push.
      # Example: latest
      IMAGE_TAG="$OPTARG"
      ;;
    P)
      # Defines the image name prefix to re-tag the image before pushing.
      # Example: ghcr.io/furiko-io
      NEW_IMAGE_NAME_PREFIX="$OPTARG"
      ;;
    T)
      # Defines the image tag to re-tag the image before pushing.
      # Example: latest
      NEW_IMAGE_TAG="$OPTARG"
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      exit 2
      ;;
    :)
      echo "Usage: ./push-images.sh -p IMAGE_NAME_PREFIX -t IMAGE_TAG [-P NEW_IMAGE_NAME_PREFIX] [-T NEW_IMAGE_TAG]" >&2
      exit 2
      ;;
  esac
done

NEW_IMAGE_NAME_PREFIX="${NEW_IMAGE_NAME_PREFIX:-"${IMAGE_NAME_PREFIX}"}"
NEW_IMAGE_TAG="${NEW_IMAGE_TAG:-"${IMAGE_TAG}"}"

if [[ -z "${IMAGE_NAME_PREFIX}" || -z "${IMAGE_TAG}" || -z "${NEW_IMAGE_NAME_PREFIX}" || -z "${NEW_IMAGE_TAG}" ]]
then
  echo "Usage: ./push-images.sh -p IMAGE_NAME_PREFIX -t IMAGE_TAG [-P NEW_IMAGE_NAME_PREFIX] [-T NEW_IMAGE_TAG]" >&2
  exit 2
fi

# Tag and push all images.
while IFS= read -r IMAGE; do
  SRC_IMAGE="${IMAGE_NAME_PREFIX}/${IMAGE}:${IMAGE_TAG}"
  NEW_IMAGE="${NEW_IMAGE_NAME_PREFIX}/${IMAGE}:${NEW_IMAGE_TAG}"

  # Default the target image to be the source image.
  TARGET_IMAGE="${SRC_IMAGE}"

  # If NEW_IMAGE is not equal to SRC_IMAGE, re-tag the image and push that tag instead.
  if [ "${SRC_IMAGE}" != "${NEW_IMAGE}" ]; then
    TARGET_IMAGE="${NEW_IMAGE}"

    # Check if the tag already exists on the local builder, if so, delete it.
    if [[ $(docker image ls "${TARGET_IMAGE}" -q) ]]; then
      docker rmi "${TARGET_IMAGE}"
    fi

    # Tag the new image from the source image name.
    docker tag "${SRC_IMAGE}" "${TARGET_IMAGE}"
  fi

  # Push the target image.
  docker push "${TARGET_IMAGE}"
done < ./hack/docker-images.txt
