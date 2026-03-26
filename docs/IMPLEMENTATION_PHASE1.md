# Sera 实现文档 - Phase 1

## 已完成的基础设施

### 1. 数据库迁移 (✅ 完成)

**文件**: `scripts/migrations/000002_add_deployment_tables.up.sql`

新增了 6 个核心表：

| 表名 | 用途 | 关键字段 |
|------|------|----------|
| `deployment_state_histories` | 部署状态变更历史 | deployment_id, from_state, to_state, triggered_by, agent_name |
| `ssh_recordings` | SSH 会话录制 (asciinema) | recording_url (MinIO), duration_ms, status |
| `cloud_credentials` | 云厂商凭证 (Vault 集成) | provider, vault_path, permissions |
| `deployment_templates` | 部署模板市场 | slug, template_type, tech_stack, rating |
| `natps_dlq_logs` | NATS 死信队列 | subject, error_reason, original_payload |
| `scaling_policies` | 自动扩缩容策略 | min_instances, max_instances, target_cpu_percent |

**Deployment 表扩展**:
- `current_state` - 当前状态机状态
- `state_history` - 状态历史 JSONB 数组
- `idempotency_key` - 幂等性保证

---

### 2. NATS JetStream 客户端 (✅ 完成)

**文件**: `internal/nats/client.go`

**核心功能**:
- JetStream 发布/订阅
- 流管理（Streams）
- 消费者管理（Consumers）
- Dead Letter Queue (DLQ) 支持
- 事件总线封装

**消息流**:
```
RequirementParser → code.analysis.required
CodeAnalyzer → deployment.steps.ready
DeploymentExecutor → deployment.completed / deployment.failed
Troubleshooter → diagnosis.completed
```

**DLQ 处理**:
- 自动重试失败消息
- 错误原因记录到 `natps_dlq_logs` 表
- 支持人工审查和重放

---

### 3. 统一错误包装器 (✅ 完成)

**文件**: `pkg/errors/errors.go`

**错误码分类**:
- 通用错误：`UNKNOWN`, `INVALID_ARGUMENT`, `NOT_FOUND`
- 认证/授权：`UNAUTHENTICATED`, `UNAUTHORIZED`
- Agent 相关：`AGENT_EXECUTION`, `AGENT_TIMEOUT`
- NATS 相关：`NATS_CONNECTION`, `NATS_NO_RESPONDER`
- 部署相关：`DEPLOYMENT_FAILED`, `DEPLOYMENT_ROLLBACK`
- SSH 相关：`SSH_CONNECTION`, `SSH_EXECUTION`
- 命令安全：`COMMAND_BLOCKED`, `SANDBOX_ESCAPE`

**错误严重程度**:
- `debug` - 调试信息
- `info` - 信息性错误
- `warning` - 警告（如参数验证失败）
- `error` - 一般错误
- `critical` - 严重错误（如命令安全检查失败）

**Fluent API**:
```go
errors.CommandBlocked(command, reason).
    WithContext("severity", "critical").
    WithSolution("请使用安全的命令或联系管理员")
```

---

### 4. Temporal 工作流集成 (✅ 完成)

**文件**:
- `internal/temporal/workflow.go` - 工作流定义
- `internal/temporal/state.go` - 状态机管理
- `internal/temporal/activity.go` - Activity 实现

**状态机**:
```
PENDING → ENV_PREPARING → CODE_FETCHING → DEPENDENCY_INSTALLING → BUILDING → CONFIGURING → DEPLOYING → VERIFYING → COMPLETED
                                ↓                    ↓                ↓              ↓             ↓
                            ROLLING_BACK ←─────────────────────────────────────────────────────────┘
                                ↓
                              FAILED
```

**工作流步骤**:
1. **RequirementParserActivity** - 解析 GitHub 仓库，识别项目类型
2. **CodeAnalyzerActivity** - 分析代码结构，生成部署步骤
3. **DeploymentExecutorActivity** - 执行部署（集成命令沙箱）
4. **TroubleshooterActivity** - 故障诊断（失败时触发）
5. **KnowledgeStorageActivity** - 知识入库（异步）

---

### 5. 命令安全沙箱 (✅ 完成)

**文件**: `internal/sandbox/command_sandbox.go`

**白名单机制**:
- 包管理器：`apt-get`, `yum`, `dnf`, `apk`, `npm`, `yarn`, `pip`
- 容器工具：`docker`, `docker-compose`
- 开发工具：`git`, `go`, `make`, `node`, `python`, `java`
- 系统工具：`curl`, `wget`, `ps`, `top`, `netstat`, `ping`
- 文件操作：`ls`, `cat`, `head`, `tail`, `grep`, `find`, `mkdir`, `cp`, `mv`
- 权限工具：`chmod`, `chown`（需要特殊检查）

**危险模式检测**:
| 严重程度 | 模式 | 说明 |
|----------|------|------|
| critical | `rm -rf /` | 删除根目录 |
| critical | `rm -rf *` | 删除所有文件 |
| critical | `mkfs.` | 格式化文件系统 |
| critical | `dd if=... of=/dev/sd` | 直接写磁盘 |
| critical | `:(){ :|:& };:` | Fork 炸弹 |
| high | `chmod ... 777` | 过度权限 |
| high | `chown -R root` | 提权 |
| high | `curl ... | bash` | 管道到 shell |
| medium | `cat /etc/shadow` | 读取密码文件 |
| medium | `history -c` | 清除历史 |

**特殊命令检查**:
- `rm`: 检查是否删除受保护目录（/etc, /usr, /bin 等）
- `curl/wget`: 检查 URL 是否安全（限制 http:// 外部地址）
- `chmod`: 检查是否设置 SUID/SGID 位

---

### 6. SSH 客户端集成沙箱 (✅ 完成)

**文件**:
- `internal/ssh/client.go` - SSH 客户端
- `internal/ssh/factory.go` - SSH 工厂

**集成方式**:
```go
// SSH 客户端在执行命令前先通过沙箱验证
func (s *sshClient) Execute(ctx context.Context, command string, opts ExecuteOptions) (*ExecuteResult, error) {
    // 1. 命令安全验证
    if s.sandbox != nil {
        validationResult := s.sandbox.ValidateOnly(command)
        if !validationResult.Allowed {
            return nil, fmt.Errorf("command blocked by security sandbox: %s", validationResult.Reason)
        }
    }

    // 2. 执行 SSH 命令
    // ...
}
```

**SSH 工厂模式**:
- `SSHClientFactory` - 简单工厂（使用固定配置）
- `DatabaseSSHClientFactory` - 从数据库获取服务器配置

**连接池**:
- 最大连接数：10
- 自动重连机制
- 连接复用

---

### 7. Multi-Agent 系统 (✅ 完成)

**文件**: `internal/agent/agents.go`

**4 个 Agent**:

| Agent | 职责 | 输入 | 输出 |
|-------|------|------|------|
| **RequirementParser** | 分析 GitHub 仓库，识别项目类型 | repo_url, branch | project_identity, deployment_profile, dependencies |
| **CodeAnalyzer** | 深度分析代码，生成部署配置 | repo_url, project_analysis | build_config, dockerfile, deploy_steps |
| **DeploymentExecutor** | SSH 执行部署步骤 | server_id, deploy_steps | step_results, endpoints |
| **Troubleshooter** | 故障诊断，生成修复方案 | error_log, exec_context | root_cause, remediation_plan |

**LLM 集成点**:
- RequirementParser: 分析 README 和文件树，生成结构化输出
- CodeAnalyzer: 生成 Dockerfile 和部署配置
- Troubleshooter: 根因分析，生成修复方案

---

## 架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          API Layer (Gin)                            │
│  POST /deployments  │  GET /deployments/:id  │  WS /deployments/:id │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Temporal Workflow Engine                       │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ DeploymentWorkflow                                          │   │
│  │  State: PENDING → ENV_PREPARING → ... → COMPLETED/FAILED    │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                   │
         ┌─────────────────────────┼─────────────────────────┐
         ▼                         ▼                         ▼
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│ Requirement     │      │ Code            │      │ Deployment      │
│ Parser Activity │─────▶│ Analyzer Activity│─────▶│ Executor Activity│
│ (LLM)           │      │ (LLM)           │      │ (SSH + Sandbox) │
└─────────────────┘      └─────────────────┘      └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │ SSH Client      │
                                               │ + Command       │
                                               │ Sandbox         │
                                               └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │ Target Server   │
                                               │ (Docker/Native) │
                                               └─────────────────┘

Failure Path:
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│ Deployment      │─────▶│ Troubleshooter  │─────▶│ Knowledge       │
│ Executor (Fail) │      │ Activity (LLM)  │      │ Storage (RAG)   │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

---

## 数据流

### 成功路径
```
1. User → API: POST /deployments { repo_url, server_id }
2. API → Temporal: StartDeploymentWorkflow()
3. Temporal → RequirementParser: Analyze GitHub repo
4. RequirementParser → NATS: Publish code.analysis.required
5. Temporal → CodeAnalyzer: Generate deploy config
6. CodeAnalyzer → NATS: Publish deployment.steps.ready
7. Temporal → DeploymentExecutor: Execute steps via SSH
8. DeploymentExecutor → SSH: Run commands (sandbox validated)
9. SSH → Server: Execute deployment
10. Server → SSH: Return output
11. DeploymentExecutor → Temporal: Success
12. Temporal → KnowledgeStorage: Store experience
13. Temporal → API: Workflow completed
14. API → User: Deployment success with endpoints
```

### 失败路径（带故障诊断）
```
7. Temporal → DeploymentExecutor: Execute steps
8. DeploymentExecutor → SSH: Run command
9. SSH → Server: Command fails
10. DeploymentExecutor → Temporal: Error
11. Temporal → Troubleshooter: Diagnose error
12. Troubleshooter → LLM: Analyze error log
13. Troubleshooter → Temporal: Root cause + remediation
14. Temporal → KnowledgeStorage: Store failure case
15. Temporal → API: Workflow failed with diagnosis
```

---

## 下一步 (Phase 2)

### 高优先级
1. **LLM 客户端实现** - 集成 Claude/OpenAI
2. **GitHub API 客户端** - 获取仓库元数据和文件树
3. **RAG 知识库实现** - Milvus 向量数据库集成
4. **Vault 集成** - 动态 SSH 证书管理
5. **WebSocket 实时推送** - 部署进度流式传输

### 中优先级
6. **asciinema 录制集成** - SSH 会话录制和回放
7. **MinIO 对象存储** - 存储录制文件和日志
8. **部署模板市场** - 社区模板上传和搜索
9. **自动扩缩容** - 基于 CPU/内存指标的 HPA

### 低优先级
10. **预测性故障检测** - 基于历史数据的 ML 模型
11. **多区域部署** - 跨可用区冗余
12. **成本优化** - 自动选择性价比最高的云资源

---

## 技术栈总结

| 类别 | 技术 |
|------|------|
| **API** | Gin, gorilla/websocket |
| **数据库** | PostgreSQL 15+ |
| **消息队列** | NATS JetStream |
| **工作流** | Temporal.io |
| **向量数据库** | Milvus |
| **图数据库** | Neo4j |
| **时序数据库** | ClickHouse |
| **对象存储** | MinIO |
| **秘密管理** | HashiCorp Vault |
| **SSH 录制** | asciinema |
| **AI/LLM** | Claude API / OpenAI |
| **测试** | Testcontainers |

---

## 安全特性

1. **命令沙箱** - 白名单 + 危险模式检测
2. **Vault 集成** - 动态证书，零静态密码
3. **SSH 录制** - 所有操作可审计
4. **幂等性** - 防止重复部署
5. **状态机验证** - 防止非法状态转换
6. **错误分级** - 敏感信息不泄露

---

## 监控和可观测性

1. **结构化日志** - logrus + JSON 格式
2. **审计日志** - 所有用户操作记录
3. **状态历史** - 完整的部署状态变更追踪
4. **DLQ 监控** - 失败消息告警
5. **Temporal UI** - 工作流可视化

---

## 部署拓扑

```
┌──────────────────────────────────────────────────────────────┐
│                      User's Infrastructure                    │
│                                                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Server 1  │  │   Server 2  │  │   Server 3  │          │
│  │ (Docker)    │  │ (Native)    │  │ (K8s Node)  │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└──────────────────────────────────────────────────────────────┘
                              ▲
                              │ SSH (sandbox validated)
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                     Sera Platform                           │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  API Servers (Gin, stateless, auto-scale)            │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Temporal Workers (workflow execution)               │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  NATS JetStream Cluster (3 nodes)                    │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  PostgreSQL (primary + replica)                      │   │
│  │  Milvus (vector embeddings)                          │   │
│  │  Redis (cache + sessions)                            │   │
│  │  MinIO (recordings, artifacts)                       │   │
│  │  Vault (secrets)                                     │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

## 关键设计决策

### 为什么选择 Temporal 而不是自己实现工作流引擎？
- **持久化状态** - 工作流状态自动持久化，服务器重启不丢失
- **重试内置** - 指数退避、抖动、最大重试次数
- **定时器和超时** - 原生支持
- **可观测性** - Temporal UI 可视化工作流
- **Saga 模式** - 原生支持补偿事务

### 为什么选择 NATS JetStream 而不是 Kafka？
- **轻量级** - 单个二进制，无 ZooKeeper 依赖
- **低延迟** - 微秒级消息传递
- **简单** - 配置和管理比 Kafka 简单得多
- **足够用** - JetStream 提供持久化、流、消费者

### 为什么选择命令沙箱而不是直接执行？
- **安全第一** - 防止 AI 生成危险命令
- **可解释性** - 每个被阻止的命令都有明确原因
- **可扩展** - 可以轻松添加新的检测规则
- **审计** - 所有命令执行都有记录

---

## 性能指标（预期）

| 指标 | 目标 |
|------|------|
| 部署延迟（P50） | < 30 秒 |
| 部署延迟（P99） | < 5 分钟 |
| 并发部署数 | 100+ |
| SSH 连接复用率 | > 80% |
| 命令沙箱验证延迟 | < 10ms |
| NATS 消息延迟 | < 1ms |
| Temporal 工作流启动 | < 100ms |

---

## 错误预算（SLO）

| SLI | 目标 |
|-----|------|
| 部署成功率 | > 95% |
| 故障诊断准确率 | > 80% |
| API 可用性 | > 99.9% |
| 工作流完成率 | > 99% |
| SSH 连接成功率 | > 98% |
