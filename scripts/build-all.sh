#!/usr/bin/env bash

# ---------------------------------------------------------------------
# Script Name: build-all.sh
# Description:
#   用于批量构建并推送多个服务的 Docker 镜像。
#   支持统一的镜像仓库地址传入，并对每个服务进行标准构建流程：
#     - docker build
#     - docker tag
#     - docker push
#
#   特殊服务 be-user 还会注入构建参数 CRYPTO_KEY。
#   支持错误中断：一旦某个服务构建失败，脚本会立即退出。
#   提供高亮日志输出，增强可读性。
#
# Usage:
#   ./build-all.sh <image-repo>
#
# Example:
#   ./build-all.sh registry.cn-hangzhou.aliyuncs.com/myproject
#
# Author: cc
# ---------------------------------------------------------------------

# 这个脚本主要用于打包镜像

set -e  # 一旦有命令失败就退出

# 捕获 SIGINT 信号（Ctrl+C）并退出
trap 'echo "Script interrupted."; exit 1' SIGINT

# shellcheck disable=SC2034
ds=(
  "be-banner"
  "be-calendar"
  "be-ccnu"
  "be-class"
  "be-classlist"
  "be-counter"
  "be-department"
  "be-elecprice"
  "be-feed"
  "be-grade"
  "be-infosum"
  "be-website"
  "bff"
)

imageRepo=$1

for d in "${ds[@]}"; do
  echo -e "\n\033[1;34m🔧🔧🔧 Building and pushing image for $d 🔧🔧🔧\033[0m\n"

  # shellcheck disable=SC2046
  docker build -t "$d:v1" -f "./$d/Dockerfile" .
  docker tag "$d:v1" "$imageRepo/$d:v1"
  docker push "$imageRepo/$d:v1"
done


speciald="be-user"

docker build -t "$speciald:v1" -f "./$speciald/Dockerfile" --build-arg KEY="muxiStudio123456"  .

docker tag "$speciald:v1" "$imageRepo/$speciald:v1"
docker push "$imageRepo/$speciald:v1"