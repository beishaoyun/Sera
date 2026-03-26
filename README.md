# Sera Platform

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev)
[![Next.js](https://img.shields.io/badge/Next.js-14-black.svg)](https://nextjs.org)

**AI 驱动的自动化服务器部署平台** - 输入 GitHub URL，AI 自动分析并部署到客户服务器

## 核心功能

- **服务器托管**: 客户输入服务器 root IP 和密码，平台安全托管
- **一键部署**: 输入 GitHub 项目 URL，AI 自动分析并部署
- **Multi-Agent 系统**: 4 个智能 Agent 协作完成部署
- **RAG 知识库**: 每次部署积累经验，自动进化优化
- **故障诊断**: AI 多轮对话排错，自动修复问题
- **实时进度**: WebSocket 流式传输部署进度
- **会话录制**: asciinema 录制所有 SSH 操作

## 快速开始

### 一键部署（推荐）

```bash
# 开发环境
./scripts/deploy.sh dev

# 生产环境
./scripts/deploy.sh prod

# 查看状态
./scripts/deploy.sh status
```

### 手动部署

```bash
# 1. 克隆项目
git clone https://github.com/beishaoyun/Sera.git
cd Sera

# 2. 启动依赖服务
docker compose up -d

# 3. 运行数据库迁移
go run cmd/migrator/main.go

# 4. 启动 API 服务器
go run cmd/server/main.go

# 5. 启动 Worker
go run cmd/worker/main.go

# 6. 启动前端
cd frontend && npm install && npm run dev
```

## 技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                      前端 (Next.js 14)                       │
│   Dashboard | 服务器管理 | 部署监控 | AI 对话界面               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Server (Go + Gin)                     │
│              JWT 认证 | WebSocket | 路由                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Temporal 工作流引擎                          │
│     部署工作流 | Saga 事务补偿 | 状态机管理                    │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│  Requirement     │ │    Code          │ │   Deployment     │
│  Parser Agent    │ │  Analyzer Agent  │ │ Executor Agent   │
│  (LLM + GitHub)  │ │  (LLM + Analysis)│ │  (SSH + Sandbox) │
└──────────────────┘ └──────────────────┘ └──────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    RAG 知识库系统                             │
│   Milvus(向量) | Neo4j(图) | ClickHouse(时序)               │
└─────────────────────────────────────────────────────────────┘
```

### 核心技术栈

| 类别 | 技术 |
|------|------|
| **前端** | Next.js 14, TypeScript, TailwindCSS, Radix UI |
| **后端** | Go 1.21+, Gin, pgx, Redis |
| **工作流** | Temporal.io |
| **消息** | NATS JetStream |
| **向量数据库** | Milvus |
| **图数据库** | Neo4j |
| **时序数据库** | ClickHouse |
| **对象存储** | MinIO |
| **密钥管理** | HashiCorp Vault |
| **SSH 录制** | asciinema |
| **AI/LLM** | Anthropic Claude, OpenAI |

## 服务端口

| 服务 | 端口 | 凭证 |
|------|------|------|
| API Server | 8080 | - |
| Frontend | 3000 | - |
| PostgreSQL | 5432 | servermind / servermind_dev |
| Redis | 6379 | password: servermind_dev |
| Milvus | 19530 | root / Milvus |
| Neo4j | 7474/7687 | neo4j / servermind_dev |
| ClickHouse | 8123/9000 | servermind / servermind_dev |
| Temporal UI | 8233 | - |
| NATS | 4222/8222 | - |
| Vault | 8200 | token: servermind_dev_token |
| MinIO Console | 9001 | minioadmin / minioadmin |

## API 端点

### 认证
```bash
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
```

### 服务器
```bash
POST   /api/v1/servers          # 创建服务器
GET    /api/v1/servers          # 列出服务器
GET    /api/v1/servers/:id      # 获取服务器详情
DELETE /api/v1/servers/:id      # 删除服务器
POST   /api/v1/servers/:id/test # 测试连接
```

### 部署
```bash
POST /api/v1/deployments            # 创建部署
GET  /api/v1/deployments            # 列出部署
GET  /api/v1/deployments/:id        # 获取部署详情
GET  /api/v1/deployments/:id/progress # 获取进度
POST /api/v1/deployments/:id/cancel # 取消部署
```

### WebSocket
```bash
WS /ws/deployments/:id  # 实时部署进度
```

### 知识库
```bash
GET /api/v1/knowledge/search?q=error  # 搜索案例
GET /api/v1/knowledge/cases           # 列出案例
```

## 环境变量

复制 `.env.example` 到 `.env` 并配置：

```bash
# 关键配置
LLM_API_KEY=your_anthropic_api_key
GITHUB_TOKEN=your_github_token
JWT_SECRET=change_this_in_production
```

完整配置参考 `.env.example`

## 项目结构

```
Sera/
├── cmd/
│   ├── server/          # API 服务器入口
│   ├── worker/          # Worker 入口
│   └── migrator/        # 数据库迁移工具
├── internal/
│   ├── agent/           # Multi-Agent 系统
│   ├── api/             # API 服务器
│   ├── asciinema/       # SSH 录制
│   ├── auth/            # JWT 认证
│   ├── config/          # 配置加载
│   ├── database/        # 数据库层
│   ├── github/          # GitHub API 客户端
│   ├── handler/         # HTTP 处理器
│   ├── integration/     # 集成测试
│   ├── knowledge/       # RAG 知识库
│   ├── llm/             # LLM 客户端
│   ├── nats/            # NATS JetStream
│   ├── sandbox/         # 命令沙箱
│   ├── ssh/             # SSH 客户端
│   ├── temporal/        # Temporal 工作流
│   ├── vault/           # Vault 集成
│   └── websocket/       # WebSocket Hub
├── pkg/
│   ├── errors/          # 统一错误处理
│   └── models/          # 数据模型
├── scripts/
│   ├── migrations/      # 数据库迁移
│   └── deploy.sh        # 部署脚本
├── docs/
│   ├── DEPLOY.md        # 部署指南
│   ├── DEVELOPMENT.md   # 开发指南
│   └── IMPLEMENTATION_PHASE1.md
├── frontend/            # Next.js 前端
├── docker-compose.yml   # Docker 编排
└── VERSION              # 版本号
```

## 开发

### 运行测试
```bash
# 后端测试
go test -v ./...

# 前端测试
cd frontend && npm test
```

### 运行集成测试
```bash
# 需要 Docker
go test -v ./internal/integration/...
```

### 代码风格
```bash
# Go 代码格式化
go fmt ./...
go vet ./...

# Frontend
cd frontend && npm run lint
```

## 部署到生产

### Docker Compose
```bash
# 构建并启动
docker compose build
docker compose up -d

# 查看日志
docker compose logs -f api worker
```

### Kubernetes
Helm charts 即将发布。

## 监控和日志

### 健康检查
```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### Temporal UI
访问 http://localhost:8233 查看工作流执行

### NATS 监控
```bash
curl http://localhost:8222/varz
curl http://localhost:8222/connz
```

## 安全特性

1. **命令沙箱**: 白名单 + 危险模式检测
2. **Vault 集成**: 动态 SSH 证书
3. **SSH 录制**: 所有操作可审计
4. **JWT 认证**: 安全的令牌管理
5. **输入验证**: 防止注入攻击

## 贡献

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

MIT License - 详见 [LICENSE](LICENSE)

## 联系方式

- 项目地址：https://github.com/beishaoyun/Sera
- 问题反馈：https://github.com/beishaoyun/Sera/issues

## 版本历史

详见 [CHANGELOG.md](CHANGELOG.md)

当前版本：**0.3.0.0**

---

<p align="center">Made with ❤️ by the Sera Team</p>
