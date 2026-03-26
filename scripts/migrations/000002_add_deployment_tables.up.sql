-- 新增 6 个数据表 + 扩展现有表
-- Migration: add_deployment_tables
-- Date: 2026-03-26

-- ============================================================================
-- 1. deployment_state_histories - 部署状态变更历史表
-- ============================================================================
CREATE TABLE IF NOT EXISTS deployment_state_histories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    from_state VARCHAR(50),
    to_state VARCHAR(50) NOT NULL,
    triggered_by VARCHAR(50) NOT NULL, -- 'agent', 'user', 'system'
    agent_name VARCHAR(100), -- 触发变更的 Agent 名称
    context JSONB DEFAULT '{}'::jsonb, -- 上下文信息
    error_message TEXT, -- 如果有错误
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 索引优化查询
CREATE INDEX idx_deployment_state_histories_deployment_id ON deployment_state_histories(deployment_id);
CREATE INDEX idx_deployment_state_histories_created_at ON deployment_state_histories(created_at DESC);
CREATE INDEX idx_deployment_state_histories_to_state ON deployment_state_histories(to_state);

-- 注释
COMMENT ON TABLE deployment_state_histories IS '部署状态变更历史记录表';
COMMENT ON COLUMN deployment_state_histories.triggered_by IS '触发来源：agent, user, system';
COMMENT ON COLUMN deployment_state_histories.context IS '状态变更时的上下文信息 (JSON)';

-- ============================================================================
-- 2. ssh_recordings - SSH 会话录制表
-- ============================================================================
CREATE TABLE IF NOT EXISTS ssh_recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    session_id VARCHAR(100) NOT NULL, -- SSH 会话唯一标识
    recording_url TEXT NOT NULL, -- MinIO 对象路径 (asciinema .cast 格式)
    recording_size_bytes BIGINT DEFAULT 0, -- 文件大小
    duration_ms BIGINT DEFAULT 0, -- 录制时长 (毫秒)
    status VARCHAR(50) DEFAULT 'recording', -- recording, completed, failed
    metadata JSONB DEFAULT '{}'::jsonb, -- 元数据 (终端尺寸、环境等)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    CONSTRAINT ssh_recordings_status_check CHECK (status IN ('recording', 'completed', 'failed'))
);

-- 索引优化
CREATE INDEX idx_ssh_recordings_deployment_id ON ssh_recordings(deployment_id);
CREATE INDEX idx_ssh_recordings_server_id ON ssh_recordings(server_id);
CREATE INDEX idx_ssh_recordings_status ON ssh_recordings(status);

-- 注释
COMMENT ON TABLE ssh_recordings IS 'SSH 会话录制表 (asciinema 格式)';
COMMENT ON COLUMN ssh_recordings.recording_url IS 'MinIO 存储路径，格式：recordings/{deployment_id}/{session_id}.cast';
COMMENT ON COLUMN ssh_recordings.metadata IS '元数据：{"terminal_cols": 80, "terminal_rows": 24, "env": "production"}';

-- ============================================================================
-- 3. cloud_credentials - 云凭证管理表 (Vault 引用)
-- ============================================================================
CREATE TABLE IF NOT EXISTS cloud_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'aws', 'digitalocean', 'gcp', 'azure'
    credential_name VARCHAR(100) NOT NULL, -- 用户自定义名称
    vault_path TEXT NOT NULL, -- HashiCorp Vault 路径
    vault_secret_id VARCHAR(255), -- Vault Secret ID (加密存储)

    -- 凭证元数据
    provider_account_id VARCHAR(100), -- 云厂商账户 ID
    provider_project_id VARCHAR(100), -- GCP 项目 ID 等
    regions TEXT[], -- 可用区域列表

    -- 权限范围
    permissions JSONB DEFAULT '{}'::jsonb, -- {"ec2": ["read", "write"], "s3": ["read"]}

    -- 状态管理
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    last_rotated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ, -- 凭证过期时间

    -- 审计
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT cloud_credentials_unique_name UNIQUE (user_id, provider, credential_name),
    CONSTRAINT cloud_credentials_provider_check CHECK (provider IN ('aws', 'digitalocean', 'gcp', 'azure'))
);

-- 索引优化
CREATE INDEX idx_cloud_credentials_user_id ON cloud_credentials(user_id);
CREATE INDEX idx_cloud_credentials_provider ON cloud_credentials(provider);
CREATE INDEX idx_cloud_credentials_is_active ON cloud_credentials(is_active) WHERE is_active = true;

-- 注释
COMMENT ON TABLE cloud_credentials IS '云厂商凭证管理表 (Vault 集成)';
COMMENT ON COLUMN cloud_credentials.vault_path IS 'Vault 中的密钥路径，如：secret/data/cloud/aws/prod';
COMMENT ON COLUMN cloud_credentials.permissions IS '权限范围定义，用于限制 Agent 可执行的操作';

-- ============================================================================
-- 4. deployment_templates - 部署模板表 (模板市场)
-- ============================================================================
CREATE TABLE IF NOT EXISTS deployment_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 基本信息
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(200) UNIQUE NOT NULL, -- URL 友好的唯一标识
    description TEXT,
    long_description TEXT, -- Markdown 格式

    -- 模板来源
    repo_url TEXT NOT NULL, -- GitHub 仓库地址
    repo_owner VARCHAR(100) NOT NULL,
    repo_name VARCHAR(100) NOT NULL,
    template_path VARCHAR(500), -- 仓库内路径 (单仓库多模板场景)
    commit_hash VARCHAR(64), -- 锁定的 commit

    -- 分类
    template_type VARCHAR(50) NOT NULL DEFAULT 'community', -- 'official', 'community', 'premium'
    category VARCHAR(100), -- 'web', 'database', 'cache', 'monitoring', etc.
    tags TEXT[], -- 标签数组

    -- 技术栈
    tech_stack JSONB DEFAULT '{}'::jsonb, -- {"language": "node", "framework": "nextjs", "db": "postgresql"}
    os_compatibility TEXT[], -- 兼容的操作系统 ['ubuntu-22.04', 'debian-11']

    -- 部署配置
    dockerfile_included BOOLEAN DEFAULT false,
    docker_compose_included BOOLEAN DEFAULT false,
    k8s_manifest_included BOOLEAN DEFAULT false,

    -- 资源需求
    min_cpu_cores DECIMAL(4,2) DEFAULT 0.5,
    min_memory_mb INT DEFAULT 512,
    min_disk_gb INT DEFAULT 5,

    -- 社区指标
    rating_avg DECIMAL(3,2) DEFAULT 0.00, -- 平均评分 0-5
    rating_count INT DEFAULT 0,
    download_count INT DEFAULT 0,
    success_count INT DEFAULT 0, -- 成功部署次数
    failure_count INT DEFAULT 0,

    -- 状态
    is_published BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false, -- 官方验证
    verified_at TIMESTAMPTZ,

    -- 作者信息
    author_id UUID REFERENCES users(id),
    author_name VARCHAR(100),

    -- 元数据
    version VARCHAR(50) DEFAULT '1.0.0',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT deployment_templates_type_check CHECK (template_type IN ('official', 'community', 'premium'))
);

-- 索引优化
CREATE INDEX idx_deployment_templates_slug ON deployment_templates(slug);
CREATE INDEX idx_deployment_templates_category ON deployment_templates(category);
CREATE INDEX idx_deployment_templates_tags ON deployment_templates USING GIN(tags);
CREATE INDEX idx_deployment_templates_tech_stack ON deployment_templates USING GIN(tech_stack);
CREATE INDEX idx_deployment_templates_is_published ON deployment_templates(is_published) WHERE is_published = true;
CREATE INDEX idx_deployment_templates_rating ON deployment_templates(rating_avg DESC) WHERE is_published = true;

-- 注释
COMMENT ON TABLE deployment_templates IS '部署模板表 (模板市场)';
COMMENT ON COLUMN deployment_templates.slug IS 'URL 友好的唯一标识，如：nextjs-blog-wordpress';
COMMENT ON COLUMN deployment_templates.tech_stack IS '技术栈定义：{"language": "node", "framework": "nextjs", "db": "postgresql"}';
COMMENT ON COLUMN deployment_templates.template_type IS '模板类型：official(官方), community(社区), premium(付费)';

-- ============================================================================
-- 5. natps_dlq_logs - NATS Dead Letter Queue 日志表
-- ============================================================================
CREATE TABLE IF NOT EXISTS natps_dlq_logs (
    id BIGSERIAL PRIMARY KEY,

    -- 消息信息
    subject VARCHAR(255) NOT NULL, -- NATS 主题
    message_id VARCHAR(100) NOT NULL, -- 消息唯一 ID
    sequence_number BIGINT, -- JetStream 序列号

    -- 错误信息
    error_reason TEXT NOT NULL, -- 进入 DLQ 的原因
    error_code VARCHAR(50), -- 错误码

    -- 原始消息
    original_payload JSONB NOT NULL, -- 原始消息体
    headers JSONB DEFAULT '{}'::jsonb, -- 消息头

    -- 重试信息
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    last_retry_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ,

    -- 处理状态
    status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'processing', 'resolved', 'discarded'
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT,

    -- 元数据
    source_agent VARCHAR(100), -- 发送消息的 Agent
    target_agent VARCHAR(100), -- 目标 Agent
    correlation_id VARCHAR(100), -- 关联 ID (用于追踪请求/响应)

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT natps_dlq_logs_status_check CHECK (status IN ('pending', 'processing', 'resolved', 'discarded'))
);

-- 索引优化
CREATE INDEX idx_natps_dlq_logs_subject ON natps_dlq_logs(subject);
CREATE INDEX idx_natps_dlq_logs_status ON natps_dlq_logs(status);
CREATE INDEX idx_natps_dlq_logs_created_at ON natps_dlq_logs(created_at DESC);
CREATE INDEX idx_natps_dlq_logs_source_agent ON natps_dlq_logs(source_agent);
CREATE INDEX idx_natps_dlq_logs_correlation_id ON natps_dlq_logs(correlation_id);

-- 注释
COMMENT ON TABLE natps_dlq_logs IS 'NATS Dead Letter Queue 日志表';
COMMENT ON COLUMN natps_dlq_logs.error_reason IS '进入 DLQ 的原因：no_responder, timeout, invalid_payload, etc.';
COMMENT ON COLUMN natps_dlq_logs.status IS '处理状态：pending(待处理), processing(处理中), resolved(已解决), discarded(已丢弃)';

-- ============================================================================
-- 6. scaling_policies - 自动伸缩策略表
-- ============================================================================
CREATE TABLE IF NOT EXISTS scaling_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,

    -- 伸缩范围
    min_instances INT NOT NULL DEFAULT 1,
    max_instances INT NOT NULL DEFAULT 10,
    desired_instances INT DEFAULT 1, -- 期望实例数

    -- 触发指标
    target_cpu_percent INT DEFAULT 80, -- 目标 CPU 使用率
    target_memory_percent INT DEFAULT 80, -- 目标内存使用率
    target_request_per_second INT, -- 目标 RPS (可选)
    target_latency_ms INT, -- 目标延迟 (可选)

    -- 冷却时间
    scale_up_cooldown_seconds INT DEFAULT 60, -- 扩容冷却时间
    scale_down_cooldown_seconds INT DEFAULT 300, -- 缩容冷却时间

    -- 定时伸缩 (可选)
    schedule_enabled BOOLEAN DEFAULT false,
    schedule_cron VARCHAR(100), -- Cron 表达式
    schedule_min_instances INT, -- 定时伸缩的最小实例数
    schedule_max_instances INT, -- 定时伸缩的最大实例数

    -- 状态
    is_active BOOLEAN DEFAULT true,
    last_scale_at TIMESTAMPTZ,
    last_scale_reason VARCHAR(255),
    last_scale_from INT,
    last_scale_to INT,

    -- 统计
    total_scale_ups INT DEFAULT 0,
    total_scale_downs INT DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT scaling_policy_instances_check CHECK (min_instances <= max_instances),
    CONSTRAINT scaling_policy_cpu_check CHECK (target_cpu_percent >= 0 AND target_cpu_percent <= 100),
    CONSTRAINT scaling_policy_memory_check CHECK (target_memory_percent >= 0 AND target_memory_percent <= 100)
);

-- 索引优化
CREATE INDEX idx_scaling_policies_deployment_id ON scaling_policies(deployment_id);
CREATE INDEX idx_scaling_policies_is_active ON scaling_policies(is_active) WHERE is_active = true;

-- 注释
COMMENT ON TABLE scaling_policies IS '自动伸缩策略表';
COMMENT ON COLUMN scaling_policies.schedule_cron IS '定时伸缩 Cron 表达式，如：0 9 * * 1-5 (工作日 9 点)';
COMMENT ON COLUMN scaling_policies.is_active IS '策略是否启用';

-- ============================================================================
-- 7. 扩展 Deployments 表 - 新增状态字段
-- ============================================================================

-- 添加状态管理字段
ALTER TABLE deployments
ADD COLUMN IF NOT EXISTS current_state VARCHAR(50) DEFAULT 'PENDING',
ADD COLUMN IF NOT EXISTS state_history JSONB DEFAULT '[]'::jsonb, -- 状态变更历史快照
ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(100) UNIQUE; -- 幂等性键

-- 索引优化
CREATE INDEX IF NOT EXISTS idx_deployments_current_state ON deployments(current_state);
CREATE INDEX IF NOT EXISTS idx_deployments_idempotency_key ON deployments(idempotency_key);

-- 注释
COMMENT ON COLUMN deployments.current_state IS '当前状态机状态';
COMMENT ON COLUMN deployments.state_history IS '状态变更历史快照 (JSON 数组)';
COMMENT ON COLUMN deployments.idempotency_key IS '幂等性键，防止重复部署';

-- ============================================================================
-- Migration Info
-- ============================================================================
DO $$
BEGIN
    RAISE NOTICE 'Migration completed: Added 6 new tables for deployment management';
    RAISE NOTICE '  - deployment_state_histories: State change history';
    RAISE NOTICE '  - ssh_recordings: SSH session recordings (asciinema)';
    RAISE NOTICE '  - cloud_credentials: Cloud provider credentials (Vault)';
    RAISE NOTICE '  - deployment_templates: Deployment templates (marketplace)';
    RAISE NOTICE '  - natps_dlq_logs: NATS Dead Letter Queue logs';
    RAISE NOTICE '  - scaling_policies: Auto-scaling policies';
    RAISE NOTICE '  - deployments: Extended with state management fields';
END $$;
