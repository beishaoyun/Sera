package models

import (
	"time"

	"github.com/google/uuid"
)

// User 用户模型
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password_hash"` // 永远不序列化到 JSON
	Name      string    `json:"name" db:"name"`
	Avatar    string    `json:"avatar,omitempty" db:"avatar"`
	Tier      string    `json:"tier" db:"tier"` // free, pro, team, enterprise
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"-" db:"deleted_at"`

	// 配额
	MaxServers      int `json:"max_servers" db:"max_servers"`
	MaxDeployments  int `json:"max_deployments" db:"max_deployments"`
	MaxConcurrent   int `json:"max_concurrent" db:"max_concurrent"`
}

// Server 用户托管的服务器
type Server struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Name      string     `json:"name" db:"name"`

	// 连接信息（加密存储）
	Host      string     `json:"host" db:"host"`
	Port      int        `json:"port" db:"port"`
	Username  string     `json:"username" db:"username"` // 通常是 root

	// 凭据（通过 Vault 管理，此处仅存储引用）
	CredentialID string  `json:"-" db:"credential_id"`

	// 服务器信息
	OS         string     `json:"os,omitempty" db:"os"`
	OSVersion  string     `json:"os_version,omitempty" db:"os_version"`
	Kernel     string     `json:"kernel,omitempty" db:"kernel"`
	CPUCores   int        `json:"cpu_cores,omitempty" db:"cpu_cores"`
	MemoryGB   int        `json:"memory_gb,omitempty" db:"memory_gb"`
	DiskGB     int        `json:"disk_gb,omitempty" db:"disk_gb"`

	// 状态
	Status     string     `json:"status" db:"status"` // online, offline, error
	LastSeen   *time.Time `json:"last_seen,omitempty" db:"last_seen"`

	// 元数据
	Tags       []string   `json:"tags,omitempty" db:"tags"` // PostgreSQL array
	Notes      string     `json:"notes,omitempty" db:"notes"`

	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// ServerStatus 服务器状态
type ServerStatus string

const (
	ServerStatusOnline  ServerStatus = "online"
	ServerStatusOffline ServerStatus = "offline"
	ServerStatusError   ServerStatus = "error"
)

// Deployment 部署记录
type Deployment struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	ServerID       uuid.UUID  `json:"server_id" db:"server_id"`
	ProjectID      uuid.UUID  `json:"project_id" db:"project_id"`

	// 项目信息
	ProjectName    string     `json:"project_name" db:"project_name"`
	RepoURL        string     `json:"repo_url" db:"repo_url"`
	Branch         string     `json:"branch" db:"branch"`
	CommitHash     string     `json:"commit_hash,omitempty" db:"commit_hash"`

	// 部署状态
	Status         string     `json:"status" db:"status"` // pending, running, completed, failed, rolling_back
	State          string     `json:"state" db:"state"`   // 详细状态机状态

	// 执行信息
	StartedAt      *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	Duration       *int64     `json:"duration_ms,omitempty" db:"duration_ms"`

	// 结果
	Result         *DeployResult `json:"result,omitempty" db:"result"`
	ErrorMessage   string     `json:"error_message,omitempty" db:"error_message"`

	// 工作流
	WorkflowID     string     `json:"workflow_id,omitempty" db:"workflow_id"`

	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// DeployResult 部署结果
type DeployResult struct {
	Success       bool              `json:"success"`
	Endpoints     []ServiceEndpoint `json:"endpoints,omitempty"`
	Steps         []DeployStep      `json:"steps,omitempty"`
	Errors        []DeployError     `json:"errors,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

// ServiceEndpoint 服务端点
type ServiceEndpoint struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Purpose  string `json:"purpose"`
	Path     string `json:"path,omitempty"`
}

// DeployStep 部署步骤
type DeployStep struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // pending, running, success, failed, skipped
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Duration    int64     `json:"duration_ms,omitempty"`
	Logs        string    `json:"logs,omitempty"`
}

// DeployError 部署错误
type DeployError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Severity  string `json:"severity"` // critical, error, warning
	Step      string `json:"step,omitempty"`
	Solution  string `json:"solution,omitempty"`
}

// Project 项目（GitHub 仓库）
type Project struct {
	ID              uuid.UUID `json:"id" db:"id"`
	UserID          uuid.UUID `json:"user_id" db:"user_id"`
	Name            string    `json:"name" db:"name"`
	RepoURL         string    `json:"repo_url" db:"repo_url"`
	RepoOwner       string    `json:"repo_owner" db:"repo_owner"`
	RepoName        string    `json:"repo_name" db:"repo_name"`
	DefaultBranch   string    `json:"default_branch" db:"default_branch"`

	// 项目分析结果
	Language        string    `json:"language,omitempty" db:"language"`
	FrameWork       string    `json:"framework,omitempty" db:"framework"`
	DeployType      string    `json:"deploy_type" db:"deploy_type"` // docker, native, k8s
	HasDockerfile   bool      `json:"has_dockerfile" db:"has_dockerfile"`
	HasDockerCompose bool     `json:"has_docker_compose" db:"has_docker_compose"`

	// 资源需求
	MinCPU        float64 `json:"min_cpu_cores" db:"min_cpu_cores"`
	MinMemory     int     `json:"min_memory_mb" db:"min_memory_mb"`
	MinDisk       int     `json:"min_disk_gb" db:"min_disk_gb"`

	// 端口
	ExposedPorts  []int   `json:"exposed_ports,omitempty" db:"exposed_ports"`

	// 环境变量
	EnvVars       []EnvVar `json:"env_vars,omitempty" db:"env_vars"`

	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// EnvVar 环境变量定义
type EnvVar struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Sensitive   bool   `json:"sensitive"` // 是否敏感，需要加密存储
}

// KnowledgeCase 知识库案例
type KnowledgeCase struct {
	ID          uuid.UUID `json:"id" db:"id"`

	// 案例类型
	CaseType    string `json:"case_type" db:"case_type"` // success, failure

	// 环境指纹
	OS          string `json:"os" db:"os"`
	OSVersion   string `json:"os_version" db:"os_version"`
	TechStack   string `json:"tech_stack" db:"tech_stack"`
	Runtime     string `json:"runtime" db:"runtime"`

	// 问题描述（失败案例）
	ErrorType   string `json:"error_type,omitempty" db:"error_type"`
	ErrorLog    string `json:"error_log,omitempty" db:"error_log"`
	RootCause   string `json:"root_cause,omitempty" db:"root_cause"`

	// 解决方案
	Solution    string `json:"solution,omitempty" db:"solution"`
	Commands    []string `json:"commands,omitempty" db:"commands"`

	// 向量嵌入（Milvus 存储）
	Embedding   []float32 `json:"-" db:"-"`

	// 统计
	SuccessCount int `json:"success_count" db:"success_count"`
	FailureCount int `json:"failure_count" db:"failure_count"`

	// 质量评分
	QualityScore float64 `json:"quality_score" db:"quality_score"`

	// 状态
	IsActive    bool      `json:"is_active" db:"is_active"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty" db:"verified_at"`

	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// AuditLog 审计日志
type AuditLog struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Action     string    `json:"action" db:"action"`
	Resource   string    `json:"resource" db:"resource"`
	ResourceID uuid.UUID `json:"resource_id" db:"resource_id"`

	// 详情
	Details    string    `json:"details,omitempty" db:"details"`
	IPAddress  string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  string    `json:"user_agent,omitempty" db:"user_agent"`

	// 结果
	Success    bool      `json:"success" db:"success"`
	Error      string    `json:"error,omitempty" db:"error"`

	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
