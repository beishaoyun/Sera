# Sera

AI 驱动的自动化服务器部署平台 - 客户托管服务器，AI 自动部署 GitHub 项目

## 核心功能

- **服务器托管**: 客户输入服务器 root IP 和密码，平台安全托管
- **一键部署**: 输入 GitHub 项目 URL，AI 自动分析并部署
- **Multi-Agent 系统**: 4 个智能 Agent 协作完成部署
- **RAG 知识库**: 每次部署积累经验，自动进化优化
- **故障诊断**: AI 多轮对话排错，自动修复问题

## 技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                      前端 (Next.js 14)                       │
│   Dashboard | 服务器管理 | 部署监控 | AI 对话界面               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway (Kong)                      │
│                    认证 | 限流 | 路由                         │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│   User Service   │ │  Server Service  │ │ Deployment Svc   │
│   (Go + Gin)     │ │   (Go + Gin)     │ │   (Go + Gin)     │
└──────────────────┘ └──────────────────┘ └──────────────────┘
              │               │               │
              └───────────────┼───────────────┘
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
└──────────────────┘ └──────────────────┘ └──────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    RAG 知识库系统                             │
│   Milvus(向量) | Neo4j(图) | ClickHouse(时序)               │
└─────────────────────────────────────────────────────────────┘
```

## 快速开始

### 开发环境

```bash
# 1. 克隆项目
git clone https://github.com/beishaoyun/Sera.git
cd Sera

# 2. 启动依赖服务 (Docker Compose)
docker compose up -d postgres redis milvus neo4j clickhouse temporal

# 3. 初始化数据库
go run cmd/migrator/main.go

# 4. 启动后端服务
go run cmd/server/main.go

# 5. 启动 Worker (处理部署任务)
go run cmd/worker/main.go

# 6. 启动前端
cd frontend && npm install && npm run dev
```

### 访问

- 前端控制台：http://localhost:3000
- API 文档：http://localhost:8080/swagger

## 核心 Agent

| Agent | 职责 | 状态机 |
|-------|------|--------|
| RequirementParser | 解析 GitHub 项目，提取部署需求 | 输入→分析→输出 |
| CodeAnalyzer | 深度分析代码结构，生成部署拓扑 | 扫描→推断→生成 |
| DeploymentExecutor | 在目标服务器执行部署命令 | 准备→执行→验证 |
| Troubleshooter | 故障诊断，多轮对话排错 | 诊断→检索→修复 |

## 项目结构

```
Sera/
├── cmd/                    # 可执行程序入口
│   ├── server/            # 主 API 服务
│   ├── worker/            # 异步任务 Worker
│   └── migrator/          # 数据库迁移工具
├── internal/              # 内部包（外部不可引用）
│   ├── agent/             # Multi-Agent 系统
│   ├── api/               # HTTP API handlers
│   ├── auth/              # 认证授权
│   ├── database/          # 数据库访问层
│   ├── knowledge/         # RAG 知识库
│   ├── scheduler/         # 任务调度
│   ├── ssh/               # SSH 连接管理
│   └── workflow/          # Temporal 工作流
├── pkg/                   # 公共包
│   ├── models/            # 数据模型
│   ├── rag/               # RAG 工具
│   ├── utils/             # 通用工具
│   └── validator/         # 参数校验
├── frontend/              # Next.js 前端
│   ├── app/               # App Router 页面
│   ├── components/        # React 组件
│   └── lib/               # 工具库
├── configs/               # 配置文件
│   ├── k8s/               # Kubernetes 部署
│   ├── docker/            # Docker 配置
│   └── prometheus/        # 监控配置
├── scripts/               # 脚本工具
├── tests/                 # 测试用例
└── docs/                  # 文档
```

## 商业模式

| 层级 | 价格 | 服务器 | 部署次数 | 功能 |
|------|------|--------|----------|------|
| 免费 | $0 | 3 台 | 10 次/月 | 基础 AI 诊断 |
| Pro | $49/月 | 20 台 | 无限 | 高级 AI 诊断 + 私有知识库 |
| Team | $199/月 | 100 台 | 无限 | 多 Agent 并行 + API 访问 |
| Enterprise | 定制 | 无限 | 无限 | 私有化部署 + 专属模型 |

## License

MIT License
