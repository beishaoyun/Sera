#!/bin/bash

# ServerMind 快速启动脚本

set -e

echo "========================================"
echo "  ServerMind 快速启动"
echo "========================================"
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "错误：未检测到 Docker，请先安装 Docker"
    exit 1
fi

# 检查 Go
if ! command -v go &> /dev/null; then
    echo "错误：未检测到 Go，请先安装 Go 1.21+"
    exit 1
fi

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "警告：未检测到 Node.js，前端将无法运行"
fi

echo "1. 启动基础服务 (PostgreSQL, Redis)..."
docker-compose up -d postgres redis

echo "等待服务启动..."
sleep 5

echo "2. 运行数据库迁移..."
go run cmd/migrator/main.go

echo "3. 启动 API 服务器..."
go run cmd/server/main.go &
API_PID=$!

echo "4. 启动 Worker..."
go run cmd/worker/main.go &
WORKER_PID=$!

echo ""
echo "========================================"
echo "  ServerMind 已启动!"
echo "========================================"
echo ""
echo "  API 服务器：http://localhost:8080"
echo "  健康检查：http://localhost:8080/health"
echo ""
echo "按 Ctrl+C 停止服务"
echo ""

# 等待中断信号
trap "kill $API_PID $WORKER_PID 2>/dev/null; echo '服务已停止'; exit 0" INT

# 保持脚本运行
wait
