# ServerMind 开发指南

## 环境要求

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- Git

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/servermind/aixm.git
cd aixm
```

### 2. 安装依赖

```bash
# Go 依赖
go mod download

# 前端依赖
cd frontend && npm install
```

### 3. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，修改配置
```

### 4. 启动服务

#### 方式一：使用脚本

```bash
./scripts/dev/start.sh
```

#### 方式二：使用 Make

```bash
make dev
```

#### 方式三：手动启动

```bash
# 启动数据库
docker-compose up -d postgres redis

# 运行迁移
go run cmd/migrator/main.go

# 启动 API
go run cmd/server/main.go

# 启动 Worker
go run cmd/worker/main.go
```

### 5. 启动前端

```bash
cd frontend
npm run dev
```

访问 http://localhost:3000

## 项目结构

```
aixm/
├── cmd/                    # 可执行程序
│   ├── server/            # API 服务入口
│   ├── worker/            # Worker 入口
│   └── migrator/          # 数据库迁移
├── internal/              # 内部包
│   ├── agent/             # Multi-Agent 系统
│   ├── api/               # HTTP API
│   ├── auth/              # 认证授权
│   ├── database/          # 数据库层
│   ├── knowledge/         # RAG 知识库
│   ├── ssh/               # SSH 连接
│   └── workflow/          # 工作流引擎
├── pkg/                   # 公共包
│   ├── models/            # 数据模型
│   ├── rag/               # RAG 工具
│   └── utils/             # 通用工具
├── frontend/              # Next.js 前端
│   ├── app/               # 页面
│   ├── components/        # 组件
│   └── lib/               # 工具库
├── configs/               # 配置文件
├── scripts/               # 脚本
└── tests/                 # 测试
```

## API 文档

### 认证

- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/refresh` - 刷新令牌

### 服务器

- `GET /api/v1/servers` - 列出服务器
- `POST /api/v1/servers` - 创建服务器
- `GET /api/v1/servers/:id` - 获取服务器详情
- `POST /api/v1/servers/:id/connect` - 连接服务器

### 部署

- `GET /api/v1/deployments` - 列出部署
- `POST /api/v1/deployments` - 创建部署
- `GET /api/v1/deployments/:id` - 获取部署详情
- `POST /api/v1/deployments/:id/cancel` - 取消部署

## 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/agent/...

# 带覆盖率
go test -cover ./...
```

## 构建

```bash
# 编译二进制
make build

# Docker 构建
docker build -t servermind .
```

## 部署

### Docker Compose

```bash
docker-compose up -d
```

### Kubernetes

```bash
kubectl apply -f configs/k8s/
```

## 故障排查

### 数据库连接失败

```bash
# 检查数据库是否运行
docker-compose ps

# 查看数据库日志
docker-compose logs postgres
```

### 端口占用

修改 `.env` 中的端口配置：

```
SERVER_PORT=8081  # 修改 API 端口
```

## 贡献

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## License

MIT License
