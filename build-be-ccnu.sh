#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-ccnu.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-ccnu 🔧🔧🔧\033[0m\n"

docker build -t "be-ccnu:v1" -f "./be-ccnu/Dockerfile" .
docker tag "be-ccnu:v1" "$imageRepo/be-ccnu:v1"
docker push "$imageRepo/be-ccnu:v1"


