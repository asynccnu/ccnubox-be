#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-library.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-library 🔧🔧🔧\033[0m\n"

docker build -t "be-library:v1" -f "./be-library/Dockerfile" .
docker tag "be-library:v1" "$imageRepo/be-library:v1"
docker push "$imageRepo/be-library:v1"
