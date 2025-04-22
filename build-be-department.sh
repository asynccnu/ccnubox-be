#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-department.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-department 🔧🔧🔧\033[0m\n"

docker build -t "be-department:v1" -f "./be-department/Dockerfile" .
docker tag "be-department:v1" "$imageRepo/be-department:v1"
docker push "$imageRepo/be-department:v1"
