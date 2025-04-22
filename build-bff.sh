#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-bff.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for bff 🔧🔧🔧\033[0m\n"

docker build -t "bff:v1" -f "./bff/Dockerfile" .
docker tag "bff:v1" "$imageRepo/bff:v1"
docker push "$imageRepo/bff:v1"
