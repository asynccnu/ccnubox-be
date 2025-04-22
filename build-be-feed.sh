#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1

if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-feed.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-feed 🔧🔧🔧\033[0m\n"

docker build -t "be-feed:v1" -f "./be-feed/Dockerfile" .
docker tag "be-feed:v1" "$imageRepo/be-feed:v1"
docker push "$imageRepo/be-feed:v1"
