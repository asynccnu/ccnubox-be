#!/bin/bash

set -e

SERVICES_YAML="deployment/docker/services.yaml"
BUILD_SCRIPT_DIR="scripts"


# 1. 将所有命令行参数存入数组
TARGET_SERVICES=("$@")

# 2. 判断数组是否为空
if [ ${#TARGET_SERVICES[@]} -eq 0 ]; then
    echo "提示：未指定服务，执行全量构建与启动..."

    # 执行全量构建脚本
    if [ -f "${BUILD_SCRIPT_DIR}/build-all.sh" ]; then
        bash "${BUILD_SCRIPT_DIR}/build-all.sh"
    else
        echo "❌ 错误: 未找到 ${BUILD_SCRIPT_DIR}/build-all.sh"
        exit 1
    fi

    # 启动所有服务
    echo "🚀 正在启动所有服务..."
    docker compose -f "$SERVICES_YAML" up -d

else
    echo "提示：检测到指定服务列表: ${TARGET_SERVICES[*]}"

    # 遍历输入的数组
    for service in "${TARGET_SERVICES[@]}"; do
        echo "--------------------------------------------"
        echo "🏗️  正在处理服务: ${service}"

        # 执行对应的 build 脚本
        SPECIFIC_BUILD_SCRIPT="${BUILD_SCRIPT_DIR}/build-${service}.sh"
        if [ -f "$SPECIFIC_BUILD_SCRIPT" ]; then
            bash "$SPECIFIC_BUILD_SCRIPT"
        else
            echo "⚠️  警告: 未找到脚本 $SPECIFIC_BUILD_SCRIPT，跳过构建步骤。"
        fi

        # 启动特定的容器
        echo "🚀 正在启动容器: ${service}"
        docker compose -f "$SERVICES_YAML" up -d "${service}"
    done
fi

echo "--------------------------------------------"
echo "✅ 操作完成！"