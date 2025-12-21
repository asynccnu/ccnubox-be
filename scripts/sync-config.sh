#!/usr/bin/env bash

# 目标目录（基础目录）
CONFIG_DIR="./deployment/configs"
REVERSE=false

# 处理命令行选项
while getopts "r" opt; do
  case $opt in
    r)
      REVERSE=true
      ;;
    *)
      echo "用法: $0 [-r]"
      echo "  -r : 反向同步 (从 $CONFIG_DIR 同步回各项目目录)"
      exit 1
      ;;
  esac
done

# 创建目标目录（正向同步时需要）
if [ "$REVERSE" = false ]; then
    mkdir -p "$CONFIG_DIR"
fi

# 定义 原始位置 到 部署配置名 的映射
# 注意：这里和之前一致，KEY 是项目路径，VALUE 是部署目录下的文件名
declare -A CONFIG_MAP=(
    ["be-banner/config/config-example.yaml"]="be-banner.yaml"
    ["be-calendar/config/config-example.yaml"]="be-calendar.yaml"
    ["be-ccnu/config/config-example.yaml"]="be-ccnu.yaml"
    ["be-class/configs/config-example.yaml"]="be-class.yaml"
    ["be-classlist/configs/config-example.yaml"]="be-classlist.yaml"
    ["be-counter/config/config-example.yaml"]="be-counter.yaml"
    ["be-department/config/config-example.yaml"]="be-department.yaml"
    ["be-elecprice/config/config-example.yaml"]="be-elecprice.yaml"
    ["be-feed/config/config-example.yaml"]="be-feed.yaml"
    ["be-grade/config/config-example.yaml"]="be-grade.yaml"
    ["be-infosum/config/config-example.yaml"]="be-infosum.yaml"
    ["be-library/configs/config example.yaml"]="be-library.yaml"
    ["be-proxy/config/config-example.yaml"]="be-proxy.yaml"
    ["be-user/config/config-example.yaml"]="be-user.yaml"
    ["be-website/config/config-example.yaml"]="be-website.yaml"
    ["bff/config/config-example.yaml"]="bff.yaml"
    ["be-class/configs/classrooms.json"]="classrooms.json"
)

# 遍历映射并同步
for PROJECT_PATH in "${!CONFIG_MAP[@]}"; do
    DEPLOY_FILE="$CONFIG_DIR/${CONFIG_MAP[$PROJECT_PATH]}"

    if [ "$REVERSE" = true ]; then
        # 反向同步：从部署目录 -> 项目目录
        SRC="$DEPLOY_FILE"
        DEST="$PROJECT_PATH"
    else
        # 正向同步：从项目目录 -> 部署目录
        SRC="$PROJECT_PATH"
        DEST="$DEPLOY_FILE"
    fi

    # 执行复制逻辑
    if [ -f "$SRC" ]; then
        # 如果是反向同步，确保项目里的目标子目录存在
        if [ "$REVERSE" = true ]; then
            mkdir -p "$(dirname "$DEST")"
        fi

        cp "$SRC" "$DEST"
        echo "✅ 同步完成: $SRC -> $DEST"
    else
        echo "❌ 未找到源文件: $SRC"
    fi
done

if [ "$REVERSE" = true ]; then
    echo "--- 所有配置文件已反向同步至各项目目录 ---"
else
    echo "--- 所有项目配置文件已收集至 $CONFIG_DIR ---"
fi