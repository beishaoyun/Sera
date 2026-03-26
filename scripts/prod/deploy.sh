#!/bin/bash

# ServerMind 完整部署脚本

set -e

echo "========================================"
echo "  ServerMind 部署脚本"
echo "========================================"
echo ""

cd /root/aixm

# 1. 检查依赖
echo "检查依赖..."
if ! command -v docker &> /dev/null; then
    echo "错误：未检测到 Docker"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "错误：未检测到 Go"
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo "警告：未检测到 Node.js"
fi

echo "✓ 依赖检查通过"
echo ""

# 2. 启动基础服务
echo "启动基础服务 (PostgreSQL, Redis)..."
docker-compose up -d postgres redis

echo "等待数据库启动..."
sleep 5

# 3. 下载 Go 依赖
echo "下载 Go 依赖..."
go mod download

# 4. 运行数据库迁移
echo "运行数据库迁移..."
go run cmd/migrator/main.go

# 5. 设置环境变量
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=servermind
export DB_PASSWORD=servermind_dev
export DB_NAME=servermind
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=servermind_dev
export SERVER_PORT=8080
export SERVER_MODE=debug
export JWT_SECRET=servermind_dev_secret_$(date +%s)

# 6. 启动 API 服务器（后台）
echo "启动 API 服务器..."
go run cmd/server/main.go &
API_PID=$!
echo "API 服务器 PID: $API_PID"

# 等待 API 启动
echo "等待 API 服务器启动..."
sleep 3

# 检查 API 是否启动
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✓ API 服务器已启动"
else
    echo "警告：API 服务器可能未正常启动"
fi

echo ""
echo "========================================"
echo "  ServerMind 部署完成!"
echo "========================================"
echo ""
echo "  API 地址：http://localhost:8080"
echo "  健康检查：http://localhost:8080/health"
echo ""
echo "  进程 ID: $API_PID"
echo ""
echo "停止服务：kill $API_PID"
echo ""
echo "========================================"

# 保持脚本运行
wait $API_PID
