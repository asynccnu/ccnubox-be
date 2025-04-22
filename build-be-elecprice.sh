#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1
if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-elecprice.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-elecprice 🔧🔧🔧\033[0m\n"

docker build -t "be-elecprice:v1" -f "./be-elecprice/Dockerfile" .
docker tag "be-elecprice:v1" "$imageRepo/be-elecprice:v1"
docker push "$imageRepo/be-elecprice:v1"
