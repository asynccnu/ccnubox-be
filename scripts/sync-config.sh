#!/usr/bin/env bash


# 同步各个项目的config到deployment/configs下

# 目标目录
TARGET_DIR="./deployment/configs"

# 创建目标目录（如果不存在）
mkdir -p "$TARGET_DIR"

# 定义源文件到目标文件名的映射
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

# 遍历映射并复制
for SRC in "${!CONFIG_MAP[@]}"; do
    DEST="$TARGET_DIR/${CONFIG_MAP[$SRC]}"
    if [ -f "$SRC" ]; then
        cp "$SRC" "$DEST"
        echo "已复制 $SRC -> $DEST"
    else
        echo "未找到文件: $SRC"
    fi
done

echo "所有项目配置文件复制完成！"
