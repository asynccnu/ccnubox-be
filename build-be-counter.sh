#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-counter.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-counter 🔧🔧🔧\033[0m\n"

docker build -t "be-counter:v1" -f "./be-counter/Dockerfile" .
docker tag "be-counter:v1" "$imageRepo/be-counter:v1"
docker push "$imageRepo/be-counter:v1"
