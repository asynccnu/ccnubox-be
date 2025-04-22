#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-class.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-class 🔧🔧🔧\033[0m\n"

docker build -t "be-class:v1" -f "./be-class/Dockerfile" .
docker tag "be-class:v1" "$imageRepo/be-class:v1"
docker push "$imageRepo/be-class:v1"
