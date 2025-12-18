#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1

CRYPTO_KEY=${2:"12345678"}

speciald="be-user"

echo -e "🔧🔧🔧 Building and pushing image for $speciald 🔧🔧🔧 \n"

docker build -t "$speciald:v1" -f "./$speciald/Dockerfile" --build-arg KEY="$CRYPTO_KEY" .

if [[ -n "$imageRepo" ]]; then
    echo -e "📦 Tagging and pushing $speciald to $imageRepo ...  \n"
    docker tag "$speciald:v1" "$imageRepo/$speciald:v1"
    docker push "$imageRepo/$speciald:v1"
else
    echo -e "No imageRepo provided, skipping tag & push for $speciald  \n"
fi




