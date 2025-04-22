#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1

CRYPTO_KEY=$2

if [[ -z "$imageRepo" ]]; then
  echo "Usage: ./build-be-user.sh <image-repo>"
  exit 1
fi

if [[ -z "$CRYPTO_KEY" ]]; then
  echo "Usage: ./build-be-user.sh <image-repo>"
  exit 1
fi

echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for be-user 🔧🔧🔧\033[0m\n"

docker build -t "be-user:v1" -f "./be-user/Dockerfile" --build-arg KEY="$CRYPTO_KEY" .
docker tag "be-user:v1" "$imageRepo/be-user:v1"
docker push "$imageRepo/be-user:v1"


