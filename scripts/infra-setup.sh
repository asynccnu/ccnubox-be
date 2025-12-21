#!/bin/bash
set -e

# --- 配置信息 ---
COMPOSE_FILE="deployment/docker/infra.yaml"

# etcd 配置
ETCD_CONTAINER="ccnubox-etcd"
ETCD_ENDPOINT="http://127.0.0.1:2379"
ETCD_BIN="etcdctl"
ROOT_USER="root"
ROOT_PASS="12345678"

# mysql 配置
MYSQL_CONTAINER="ccnubox-mysql"
MYSQL_ROOT_PASS="12345678"
DB_NAME="ccnubox"

echo "🚀 启动基础设施服务..."
docker compose -f ${COMPOSE_FILE} up -d

# --- 1. 修改等待逻辑 ---
echo "⏳ 等待 etcd 响应..."

# 增加 API 版本声明，确保使用 v3
until docker exec -e ETCDCTL_API=3 ${ETCD_CONTAINER} ${ETCD_BIN} \
    --endpoints=${ETCD_ENDPOINT} endpoint status >/dev/null 2>&1 || \
      docker exec -e ETCDCTL_API=3 ${ETCD_CONTAINER} ${ETCD_BIN} \
    --endpoints=${ETCD_ENDPOINT} --user=${ROOT_USER}:${ROOT_PASS} endpoint status >/dev/null 2>&1; do

  # 调试：查看 etcd 容器日志，确认它是否真的启动成功了
  if [ $((SECONDS % 10)) -eq 0 ]; then
    echo "提示：已等待 ${SECONDS}s，请检查 docker logs ${ETCD_CONTAINER}"
  fi

  echo "etcd 尚未就绪，等待中..."
  sleep 2
done

# --- 2. 判断是否已开启 auth ---
AUTH_STATUS=$(
  docker exec ${ETCD_CONTAINER} ${ETCD_BIN} \
    --endpoints=${ETCD_ENDPOINT} auth status 2>/dev/null || true
)

if echo "${AUTH_STATUS}" | grep -q "Authentication Status: false"; then
  echo "🔐 etcd 尚未开启 auth，执行初始化..."

  docker exec ${ETCD_CONTAINER} ${ETCD_BIN} user add ${ROOT_USER}:${ROOT_PASS} || true
  docker exec ${ETCD_CONTAINER} ${ETCD_BIN} role add root || true
  docker exec ${ETCD_CONTAINER} ${ETCD_BIN} user grant-role ${ROOT_USER} root || true
  docker exec ${ETCD_CONTAINER} ${ETCD_BIN} auth enable

  echo "✅ etcd auth 初始化完成"
else
  echo "🔒 etcd auth 已开启，跳过初始化"
fi

# --- 3. 再次确认 etcd（必须 auth=true） ---
echo "🔁 校验 etcd（auth=true）..."

until docker exec ${ETCD_CONTAINER} ${ETCD_BIN} \
    --endpoints=${ETCD_ENDPOINT} \
    --user=${ROOT_USER}:${ROOT_PASS} \
    endpoint health >/dev/null 2>&1; do
  echo "等待 etcd 通过 auth 校验..."
  sleep 2
done

echo "✅ etcd auth 校验通过"

# --- 4. MySQL 初始化 ---
echo "⏳ 等待 MySQL 响应...(sleep 2s)"
sleep 2

docker exec ${MYSQL_CONTAINER} mysql \
  -uroot -p"${MYSQL_ROOT_PASS}" \
  -e "CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

echo "------------------------------------------------"
echo "🎉 基础设施初始化 / 启动完成"
echo "etcd 状态: auth=true (用户: root)"
echo "MySQL 状态: 已就绪 (数据库: ${DB_NAME})"
echo "------------------------------------------------"
