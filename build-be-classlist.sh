#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-classlist.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-classlist 🔧🔧🔧\033[0m\n"

docker build -t "be-classlist:v1" -f "./be-classlist/Dockerfile" .
docker tag "be-classlist:v1" "$imageRepo/be-classlist:v1"
docker push "$imageRepo/be-classlist:v1"
