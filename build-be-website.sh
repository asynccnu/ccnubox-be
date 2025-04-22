#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-website.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-website 🔧🔧🔧\033[0m\n"

docker build -t "be-website:v1" -f "./be-website/Dockerfile" .
docker tag "be-website:v1" "$imageRepo/be-website:v1"
docker push "$imageRepo/be-website:v1"
