.PHONY: help dev build test clean migrate up down logs

# 默认目标
help:
	@echo "ServerMind 开发命令"
	@echo ""
	@echo "  make dev          - 启动开发环境（数据库 + API）"
	@echo "  make build        - 编译 Go 二进制"
	@echo "  make test         - 运行测试"
	@echo "  make clean        - 清理构建文件"
	@echo "  make migrate      - 运行数据库迁移"
	@echo "  make up           - 启动所有 Docker 服务"
	@echo "  make down         - 停止所有 Docker 服务"
	@echo "  make logs         - 查看 Docker 日志"
	@echo "  make frontend     - 启动前端开发服务器"
	@echo ""

# 开发环境
dev:
	@echo "启动开发环境..."
	docker-compose up -d postgres redis
	@echo "等待数据库启动..."
	@sleep 3
	go run cmd/migrator/main.go
	@echo "启动 API 服务器..."
	go run cmd/server/main.go

# 编译
build:
	@echo "编译服务器..."
	CGO_ENABLED=0 go build -o bin/server ./cmd/server/main.go
	@echo "编译 Worker..."
	CGO_ENABLED=0 go build -o bin/worker ./cmd/worker/main.go

# 测试
test:
	go test -v ./...

# 清理
clean:
	rm -rf bin/
	go clean

# 数据库迁移
migrate:
	go run cmd/migrator/main.go

# Docker 服务
up:
	docker-compose up -d

down:
	docker-compose down

logs:
	docker-compose logs -f

# 前端
frontend:
	cd frontend && npm run dev

# 安装依赖
install:
	go mod download
	cd frontend && npm install

# 格式化代码
fmt:
	go fmt ./...
	cd frontend && npm run lint

# 安全检查
security:
	golangci-lint run
	@echo "前端安全检查..."
	cd frontend && npm audit
