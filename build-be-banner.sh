#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-banner.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-banner 🔧🔧🔧\033[0m\n"

docker build -t "be-banner:v1" -f "./be-banner/Dockerfile" .
docker tag "be-banner:v1" "$imageRepo/be-banner:v1"
docker push "$imageRepo/be-banner:v1"
