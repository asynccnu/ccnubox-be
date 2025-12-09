#!/usr/bin/env bash

## 更新包依赖

ds=(
  "common/be-api"
  "common/pkg"
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
  "be-library"
  "be-infosum"
  "be-website"
  "be-user"
  "bff"
)

for d in "${ds[@]}"; do
    echo -e "\n===> 开始更新依赖: $d"
    if [ -d "$d" ]; then
        (cd "$d" && go mod tidy)
        echo "✅ $d 更新完成"
    else
        echo "⚠️  目录不存在: $d"
    fi
done

echo -e "\n🎉 全部依赖更新完成！"
