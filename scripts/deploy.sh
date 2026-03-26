#!/bin/bash

# Sera 快速部署脚本
# 用法：./scripts/deploy.sh [dev|prod]

set -e

ENV=${1:-dev}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "============================================"
echo "  Sera Platform - Deployment Script"
echo "  Environment: $ENV"
echo "============================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "Checking dependencies..."

    local deps=("docker" "docker-compose" "go" "node")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "$dep is not installed"
            exit 1
        fi
    done

    log_info "All dependencies found"
}

# 启动基础设施服务
start_infrastructure() {
    log_info "Starting infrastructure services..."

    cd "$PROJECT_DIR"
    docker compose up -d postgres redis milvus-standalone neo4j clickhouse temporal nats vault etcd minio

    log_info "Waiting for services to be ready..."
    sleep 30

    # 健康检查
    for service in postgres redis; do
        if ! docker compose ps "$service" | grep -q "Up"; then
            log_error "$service failed to start"
            docker compose logs "$service"
            exit 1
        fi
    done

    log_info "All infrastructure services are running"
}

# 运行数据库迁移
run_migrations() {
    log_info "Running database migrations..."

    cd "$PROJECT_DIR"
    go run cmd/migrator/main.go

    log_info "Database migrations completed"
}

# 构建后端
build_backend() {
    log_info "Building backend..."

    cd "$PROJECT_DIR"
    go build -o bin/api ./cmd/server
    go build -o bin/worker ./cmd/worker

    log_info "Backend build completed"
}

# 构建前端
build_frontend() {
    log_info "Building frontend..."

    cd "$PROJECT_DIR/frontend"

    if [ ! -d "node_modules" ]; then
        npm ci
    fi

    npm run build

    log_info "Frontend build completed"
}

# 启动开发环境
start_dev() {
    log_info "Starting development environment..."

    cd "$PROJECT_DIR"

    # 启动 API 服务器（开发模式）
    go run cmd/server/main.go &
    API_PID=$!

    # 启动 Worker
    go run cmd/worker/main.go &
    WORKER_PID=$!

    log_info "API server and worker started"
    log_info "API: http://localhost:8080"
    log_info "Temporal UI: http://localhost:8233"
    log_info "MinIO Console: http://localhost:9001"

    # 保存 PID
    echo "$API_PID" > "$PROJECT_DIR/.pids/api.pid"
    echo "$WORKER_PID" > "$PROJECT_DIR/.pids/worker.pid"
}

# 启动生产环境
start_prod() {
    log_info "Starting production environment..."

    cd "$PROJECT_DIR"

    # 构建 Docker 镜像
    docker compose build api worker

    # 启动所有服务
    docker compose up -d api worker

    log_info "Production environment started"
    log_info "API: http://localhost:8080"
}

# 停止服务
stop_services() {
    log_info "Stopping all services..."

    cd "$PROJECT_DIR"
    docker compose down

    if [ -f "$PROJECT_DIR/.pids/api.pid" ]; then
        kill $(cat "$PROJECT_DIR/.pids/api.pid") 2>/dev/null || true
        rm "$PROJECT_DIR/.pids/api.pid"
    fi

    if [ -f "$PROJECT_DIR/.pids/worker.pid" ]; then
        kill $(cat "$PROJECT_DIR/.pids/worker.pid") 2>/dev/null || true
        rm "$PROJECT_DIR/.pids/worker.pid"
    fi

    log_info "All services stopped"
}

# 显示状态
show_status() {
    echo ""
    echo "============================================"
    echo "  Sera Platform Status"
    echo "============================================"

    cd "$PROJECT_DIR"
    docker compose ps

    echo ""
    echo "Services:"
    echo "  - PostgreSQL:   localhost:5432"
    echo "  - Redis:        localhost:6379"
    echo "  - Milvus:       localhost:19530"
    echo "  - Neo4j:        localhost:7474"
    echo "  - ClickHouse:   localhost:8123"
    echo "  - Temporal:     localhost:7233 (UI: localhost:8233)"
    echo "  - NATS:         localhost:4222"
    echo "  - Vault:        localhost:8200"
    echo "  - MinIO:        localhost:9000 (Console: localhost:9001)"
    echo "  - API:          localhost:8080"
    echo ""
}

# 主函数
main() {
    case "$ENV" in
        dev)
            check_dependencies
            start_infrastructure
            run_migrations
            build_backend
            start_dev
            show_status
            ;;
        prod)
            check_dependencies
            start_infrastructure
            run_migrations
            build_backend
            build_frontend
            start_prod
            show_status
            ;;
        stop)
            stop_services
            ;;
        status)
            show_status
            ;;
        *)
            echo "Usage: $0 {dev|prod|stop|status}"
            exit 1
            ;;
    esac
}

main
