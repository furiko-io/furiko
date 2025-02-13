#!/bin/bash

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

## Simple script that generates a new kustomization.yaml and adds additional fields needed before building.
## For example, this script adds the image field necessary for substitution in kustomize.
## The purpose of this script is to create a temporary "overlay" config, so that we do not override existing
## kustomization.yaml files checked into source control.

if [ $# -ne 2 ]
then
  echo 'Usage:'
  echo '  ./generate-kustomization.sh IMAGE_NAME_PREFIX IMAGE_TAG'
  echo
  echo 'Optional environment variables:'
  echo '  DEST_DIR: Path to generate kustomization.yaml to. Default: Current working directory'
  echo '  BASE_CONFIG: Path to config path containing base kustomization.yaml. Default: ./config/default'
  echo '  KUSTOMIZE: Path to kustomize executable. Default: ./bin/kustomize'
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

# Optional environment variables
DEST_DIR="${DEST_DIR:-$(pwd)}"
KUSTOMIZE="${KUSTOMIZE:-$(pwd)/bin/kustomize}"
BASE_CONFIG="${BASE_CONFIG:-./config/default}"

# Get path to image names.
DOCKER_IMAGES="$(pwd)/hack/docker-images.txt"

# Create new kustomization.yaml to hold our substitution.
cd "${DEST_DIR}"
rm -f kustomization.yaml # Remove if exists
"${KUSTOMIZE}" create --resources "${BASE_CONFIG}"

# Add image fields.
while IFS= read -r IMAGE; do
  "${KUSTOMIZE}" edit set image "${IMAGE}=${IMAGE_NAME_PREFIX}/${IMAGE}:${IMAGE_TAG}"
done < "${DOCKER_IMAGES}"
