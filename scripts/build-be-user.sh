#!/usr/bin/env bash

set -e
trap 'echo "Script interrupted."; exit 1' SIGINT

imageRepo=$1

CRYPTO_KEY=$2

speciald="be-user"

## 这里注意需要自己配置一个加密key，否则会使用默认的key，存在安全隐患
## 本地调试可忽略
if [[ -n "$CRYPTO_KEY" ]]; then
  CRYPTO_KEY="muxiStudio123456"
fi

echo -e "🔧🔧🔧 Building and pushing image for $speciald 🔧🔧🔧 \n"

docker build -t "$speciald:v1" -f "./$speciald/Dockerfile" --build-arg KEY="$CRYPTO_KEY" .

if [[ -n "$imageRepo" ]]; then
    echo -e "📦 Tagging and pushing $speciald to $imageRepo ...  \n"
    docker tag "$speciald:v1" "$imageRepo/$speciald:v1"
    docker push "$imageRepo/$speciald:v1"
else
    echo -e "No imageRepo provided, skipping tag & push for $speciald  \n"
fi




