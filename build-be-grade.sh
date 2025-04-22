#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-grade.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-grade 🔧🔧🔧\033[0m\n"

docker build -t "be-grade:v1" -f "./be-grade/Dockerfile" .
docker tag "be-grade:v1" "$imageRepo/be-grade:v1"
docker push "$imageRepo/be-grade:v1"
