#!/usr/bin/env bash

# 启动所有服务

set -e  # 一旦有命令失败就退出

# 捕获 SIGINT 信号（Ctrl+C）并退出
trap 'echo "Script interrupted."; exit 1' SIGINT

./scripts/build-all.sh

(cd ./deployments/docker && docker compose -f docker-compose.yaml  up -d)