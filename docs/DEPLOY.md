# ServerMind 部署指南

## 快速部署（推荐 Docker）

### 方式一：使用 Docker（需要安装 Docker）

```bash
# 1. 启动所有服务
cd /root/aixm
docker compose up -d

# 2. 查看日志
docker compose logs -f api

# 3. 访问 API
# http://localhost:8080
```

### 方式二：手动部署（需要安装 Go 1.21+）

#### 1. 安装 Go

```bash
# 下载并安装 Go 1.21
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

#### 2. 安装依赖

```bash
cd /root/aixm
go mod download
```

#### 3. 启动基础服务

```bash
docker compose up -d postgres redis
```

#### 4. 运行数据库迁移

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=servermind
export DB_PASSWORD=servermind_dev
export DB_NAME=servermind

go run cmd/migrator/main.go
```

#### 5. 启动 API 服务器

```bash
export JWT_SECRET=your-secret-key-change-in-prod
export LLM_API_KEY=your-openai-api-key  # 可选，使用 LLM 功能

go run cmd/server/main.go
```

#### 6. 启动 Worker（可选）

```bash
go run cmd/worker/main.go
```

## 前端部署

```bash
cd /root/aixm/frontend

# 安装 Node.js 18+ (如未安装)
# curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
# sudo apt-get install -y nodejs

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build
npm start
```

## 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| SERVER_PORT | API 端口 | 8080 |
| SERVER_MODE | 运行模式 (debug/release) | debug |
| DB_HOST | PostgreSQL 主机 | localhost |
| DB_PORT | PostgreSQL 端口 | 5432 |
| DB_USER | 数据库用户 | servermind |
| DB_PASSWORD | 数据库密码 | servermind_dev |
| DB_NAME | 数据库名 | servermind |
| REDIS_HOST | Redis 主机 | localhost |
| REDIS_PORT | Redis 端口 | 6379 |
| JWT_SECRET | JWT 密钥 | - |
| LLM_API_KEY | OpenAI API 密钥 | - |
| LLM_PROVIDER | LLM 提供商 (openai/claude) | openai |
| LLM_MODEL | LLM 模型 | gpt-4-turbo-preview |

### .env 配置示例

```bash
# 复制示例配置
cp .env.example .env

# 编辑配置
vim .env
```

## API 测试

### 注册

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }'
```

### 登录

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### 创建部署（GitHub 项目）

```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "server_id": "xxx",
    "repo_url": "https://github.com/vercel/next.js"
  }'
```

### 创建部署（教程 URL）

```bash
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "server_id": "xxx",
    "tutorial_url": "https://juejin.cn/post/xxx"
  }'
```

### 解析内容（不部署）

```bash
curl -X POST http://localhost:8080/api/v1/parse-content \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "repo_url": "https://github.com/vercel/next.js"
  }'
```

## 故障排查

### 数据库连接失败

```bash
# 检查容器状态
docker compose ps

# 查看数据库日志
docker compose logs postgres
```

### API 无法启动

```bash
# 检查端口占用
lsof -i :8080

# 查看详细日志
go run cmd/server/main.go 2>&1 | tee server.log
```

### LLM API 失败

检查 `.env` 中的 API 密钥配置：
```bash
export LLM_API_KEY=sk-xxxxx
```

## 生产部署

### Kubernetes

```bash
kubectl apply -f configs/k8s/
```

### Systemd 服务

创建 `/etc/systemd/system/servermind.service`:

```ini
[Unit]
Description=ServerMind API
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=servermind
WorkingDirectory=/root/aixm
Environment="PATH=/usr/local/go/bin:/usr/bin"
Environment="DB_HOST=localhost"
Environment="DB_PASSWORD=servermind_dev"
ExecStart=/usr/local/go/bin/go run cmd/server/main.go
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable servermind
sudo systemctl start servermind
```
