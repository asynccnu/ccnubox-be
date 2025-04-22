#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-calendar.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-calendar 🔧🔧🔧\033[0m\n"

docker build -t "be-calendar:v1" -f "./be-calendar/Dockerfile" .
docker tag "be-calendar:v1" "$imageRepo/be-calendar:v1"
docker push "$imageRepo/be-calendar:v1"
