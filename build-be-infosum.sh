#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-infosum.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-infosum 🔧🔧🔧\033[0m\n"

docker build -t "be-infosum:v1" -f "./be-infosum/Dockerfile" .
docker tag "be-infosum:v1" "$imageRepo/be-infosum:v1"
docker push "$imageRepo/be-infosum:v1"
