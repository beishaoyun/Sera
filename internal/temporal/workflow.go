package temporal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// Config Temporal 配置
type Config struct {
	HostPort  string `mapstructure:"host_port" json:"host_port"`
	Namespace string `mapstructure:"namespace" json:"namespace"`
	TaskQueue string `mapstructure:"task_queue" json:"task_queue"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		HostPort:  "localhost:7233",
		Namespace: "default",
		TaskQueue: "servermind-deployment-queue",
	}
}

// Client Temporal 客户端包装
type Client struct {
	client    client.Client
	config    Config
	taskQueue string
}

// NewClient 创建 Temporal 客户端
func NewClient(ctx context.Context, config Config) (*Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  config.HostPort,
		Namespace: config.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to dial Temporal: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"host_port": config.HostPort,
		"namespace": config.Namespace,
	}).Info("Connected to Temporal")

	return &Client{
		client:    c,
		config:    config,
		taskQueue: config.TaskQueue,
	}, nil
}

// StartDeploymentWorkflow 启动部署工作流
func (c *Client) StartDeploymentWorkflow(ctx context.Context, input DeploymentWorkflowInput) (string, error) {
	workflowID := fmt.Sprintf("deployment-%s-%s", input.UserID, uuid.New().String()[:8])

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: c.taskQueue,
		// 工作流超时
		WorkflowExecutionTimeout: 2 * time.Hour,  // 最长执行时间
		WorkflowRunTimeout:       2 * time.Hour,  // 单次运行超时
		WorkflowTaskTimeout:      time.Minute,     // 单个任务超时
		// 重试策略
		RetryPolicy: &client.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
		// 幂等性
		WorkflowIDReusePolicy: client.WorkflowIDReusePolicyAllowDuplicateFailedOnly,
	}

	// 启动工作流
	run, err := c.client.ExecuteWorkflow(ctx, options, DeploymentWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"workflow_id": run.GetID(),
		"run_id":      run.GetRunID(),
		"user_id":     input.UserID,
		"server_id":   input.ServerID,
		"repo_url":    input.RepoURL,
	}).Info("Deployment workflow started")

	return run.GetID(), nil
}

// GetWorkflowStatus 获取工作流状态
func (c *Client) GetWorkflowStatus(ctx context.Context, workflowID string) (*WorkflowStatus, error) {
	resp, err := c.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to describe workflow: %w", err)
	}

	status := &WorkflowStatus{
		WorkflowID: workflowID,
		RunID:      resp.WorkflowExecutionInfo.RunId,
		Status:     resp.WorkflowExecutionInfo.Status.String(),
		StartTime:  resp.WorkflowExecutionInfo.StartTime,
	}

	if resp.WorkflowExecutionInfo.CloseTime != nil {
		status.CloseTime = *resp.WorkflowExecutionInfo.CloseTime
	}

	return status, nil
}

// CancelWorkflow 取消工作流
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error {
	err := c.client.CancelWorkflow(ctx, workflowID, "")
	if err != nil {
		return fmt.Errorf("failed to cancel workflow: %w", err)
	}

	logrus.WithField("workflow_id", workflowID).Info("Workflow cancelled")
	return nil
}

// SignalWorkflow 发送信号到工作流
func (c *Client) SignalWorkflow(ctx context.Context, workflowID, signalName string, arg interface{}) error {
	err := c.client.SignalWorkflow(ctx, workflowID, "", signalName, arg)
	if err != nil {
		return fmt.Errorf("failed to signal workflow: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"workflow_id": workflowID,
		"signal":      signalName,
	}).Debug("Workflow signaled")
	return nil
}

// CreateWorker 创建 Worker
func (c *Client) CreateWorker(options worker.Options) worker.Worker {
	w := worker.New(c.client, c.taskQueue, options)

	// 注册工作流
	w.RegisterWorkflow(DeploymentWorkflow)

	// 注册活动
	w.RegisterActivity(RequirementParserActivity)
	w.RegisterActivity(CodeAnalyzerActivity)
	w.RegisterActivity(DeploymentExecutorActivity)
	w.RegisterActivity(TroubleshooterActivity)
	w.RegisterActivity(KnowledgeStorageActivity)

	return w
}

// Close 关闭客户端
func (c *Client) Close() {
	c.client.Close()
	logrus.Info("Temporal client closed")
}

// ============================================================================
// 工作流定义
// ============================================================================

// DeploymentWorkflowInput 部署工作流输入
type DeploymentWorkflowInput struct {
	ID            string            `json:"id"`
	UserID        string            `json:"user_id"`
	ServerID      string            `json:"server_id"`
	ProjectID     string            `json:"project_id"`
	RepoURL       string            `json:"repo_url"`
	Branch        string            `json:"branch,omitempty"`
	CommitHash    string            `json:"commit_hash,omitempty"`
	IdempotencyKey string           `json:"idempotency_key,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// DeploymentWorkflowOutput 部署工作流输出
type DeploymentWorkflowOutput struct {
	Success     bool                   `json:"success"`
	DeploymentID string                `json:"deployment_id"`
	Endpoints   []ServiceEndpoint      `json:"endpoints,omitempty"`
	Logs        []string               `json:"logs,omitempty"`
	Duration    time.Duration          `json:"duration"`
}

// ServiceEndpoint 服务端点
type ServiceEndpoint struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Path     string `json:"path,omitempty"`
}

// WorkflowStatus 工作流状态
type WorkflowStatus struct {
	WorkflowID string    `json:"workflow_id"`
	RunID      string    `json:"run_id"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	CloseTime  time.Time `json:"close_time,omitempty"`
}

// DeploymentWorkflow 部署工作流主逻辑
func DeploymentWorkflow(ctx workflow.Context, input DeploymentWorkflowInput) (*DeploymentWorkflowOutput, error) {
	// 设置超时
	ctx = workflow.WithWorkflowTimeout(ctx, 2*time.Hour)

	// 初始化工作流状态
	state := NewDeploymentWorkflowState()
	state.WorkflowID = workflow.GetInfo(ctx).WorkflowExecution.ID
	state.Input = input

	log := workflow.GetLogger(ctx)
	log.Info("Starting deployment workflow", "input", input)

	// 步骤 1: 需求解析
	var requirementResult RequirementParserResult
	err := workflow.ExecuteActivity(ctx, RequirementParserActivity, RequirementParserInput{
		RepoURL:     input.RepoURL,
		Branch:      input.Branch,
		UserRequest: input.Metadata["user_request"],
	}).Get(ctx, &requirementResult)

	if err != nil {
		log.Error("Requirement parser failed", "error", err)
		state.RecordStepFailure("requirement_parser", err)
		return nil, err
	}
	state.RecordStepSuccess("requirement_parser", requirementResult)

	// 步骤 2: 代码分析
	var codeAnalysisResult CodeAnalyzerResult
	err = workflow.ExecuteActivity(ctx, CodeAnalyzerActivity, CodeAnalyzerInput{
		RepoURL:         input.RepoURL,
		Branch:          input.Branch,
		ProjectAnalysis: &requirementResult.Output,
	}).Get(ctx, &codeAnalysisResult)

	if err != nil {
		log.Error("Code analyzer failed", "error", err)
		state.RecordStepFailure("code_analyzer", err)
		return nil, err
	}
	state.RecordStepSuccess("code_analyzer", codeAnalysisResult)

	// 步骤 3: 部署执行
	var executionResult DeploymentExecutorResult
	err = workflow.ExecuteActivity(ctx, DeploymentExecutorActivity, DeploymentExecutorInput{
		ServerID:      input.ServerID,
		ProjectID:     input.ProjectID,
		CodeAnalysis:  &codeAnalysisResult,
		DeploySteps:   codeAnalysisResult.DeploySteps,
	}).Get(ctx, &executionResult)

	if err != nil {
		log.Error("Deployment executor failed", "error", err)
		state.RecordStepFailure("deployment_executor", err)

		// 启动故障诊断
		var troubleshootResult TroubleshooterResult
		diagErr := workflow.ExecuteActivity(ctx, TroubleshooterActivity, TroubleshooterInput{
			ErrorLog:    executionResult.ErrorMessage,
			ExecContext: &executionResult,
		}).Get(ctx, &troubleshootResult)

		if diagErr != nil {
			log.Error("Troubleshooter failed", "error", diagErr)
			return nil, err
		}

		// 如果诊断成功且有修复方案，尝试重试
		if troubleshootResult.RemediationPlan != nil {
			log.Info("Retrying with remediation plan")
			// TODO: 执行修复并重试
		}

		return nil, err
	}
	state.RecordStepSuccess("deployment_executor", executionResult)

	// 步骤 4: 知识存储（异步，不阻塞）
	workflow.Go(ctx, func(ctx workflow.Context) {
		_ = workflow.ExecuteActivity(ctx, KnowledgeStorageActivity, KnowledgeStorageInput{
			DeploymentID:   input.ID,
			Success:        true,
			ExecutionResult: &executionResult,
			CodeAnalysis:   &codeAnalysisResult,
		}).Get(ctx, nil)
	})

	// 完成
	output := &DeploymentWorkflowOutput{
		Success:      true,
		DeploymentID: input.ID,
		Endpoints:    executionResult.Endpoints,
		Duration:     state.GetDuration(),
	}

	log.Info("Deployment workflow completed", "output", output)
	return output, nil
}

// ============================================================================
// 活动定义 (桩实现 - 实际实现会调用对应的 Agent)
// ============================================================================

// RequirementParserActivity 需求解析活动
func RequirementParserActivity(ctx context.Context, input RequirementParserInput) (*RequirementParserResult, error) {
	log := logrus.WithContext(ctx)
	log.Info("Executing requirement parser activity")

	// TODO: 实际实现会调用 RequirementParser Agent
	// 这里是桩实现
	time.Sleep(2 * time.Second)

	return &RequirementParserResult{
		Output: RequirementParserOutput{
			ProjectIdentity: ProjectIdentity{
				Name:            "example-project",
				PrimaryLanguage: "Go",
			},
		},
	}, nil
}

// CodeAnalyzerActivity 代码分析活动
func CodeAnalyzerActivity(ctx context.Context, input CodeAnalyzerInput) (*CodeAnalyzerResult, error) {
	log := logrus.WithContext(ctx)
	log.Info("Executing code analyzer activity")

	// TODO: 实际实现会调用 CodeAnalyzer Agent
	time.Sleep(2 * time.Second)

	return &CodeAnalyzerResult{
		DeploySteps: []DeployStep{
			{ID: "step1", Name: "Clone repository", Description: "Clone the repository"},
			{ID: "step2", Name: "Install dependencies", Description: "Install project dependencies"},
			{ID: "step3", Name: "Build", Description: "Build the project"},
			{ID: "step4", Name: "Deploy", Description: "Deploy to server"},
		},
	}, nil
}

// DeploymentExecutorActivity 部署执行活动
func DeploymentExecutorActivity(ctx context.Context, input DeploymentExecutorInput) (*DeploymentExecutorResult, error) {
	log := logrus.WithContext(ctx)
	log.Info("Executing deployment executor activity")

	// TODO: 实际实现会调用 DeploymentExecutor Agent
	time.Sleep(5 * time.Second)

	return &DeploymentExecutorResult{
		Success: true,
		Steps: []StepExecutionResult{
			{StepID: "step1", Status: "success"},
			{StepID: "step2", Status: "success"},
			{StepID: "step3", Status: "success"},
			{StepID: "step4", Status: "success"},
		},
	}, nil
}

// TroubleshooterActivity 故障诊断活动
func TroubleshooterActivity(ctx context.Context, input TroubleshooterInput) (*TroubleshooterResult, error) {
	log := logrus.WithContext(ctx)
	log.Info("Executing troubleshooter activity")

	// TODO: 实际实现会调用 Troubleshooter Agent
	time.Sleep(2 * time.Second)

	return &TroubleshooterResult{
		Confidence: 0.8,
		RootCause:  "Example root cause",
	}, nil
}

// KnowledgeStorageActivity 知识存储活动
func KnowledgeStorageActivity(ctx context.Context, input KnowledgeStorageInput) error {
	log := logrus.WithContext(ctx)
	log.Info("Executing knowledge storage activity")

	// TODO: 实际实现会存储到 RAG 知识库
	time.Sleep(1 * time.Second)

	return nil
}

// ============================================================================
// 活动输入/输出类型
// ============================================================================

type RequirementParserInput struct {
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch,omitempty"`
	UserRequest string `json:"user_request,omitempty"`
}

type RequirementParserResult struct {
	Output RequirementParserOutput `json:"output"`
}

type RequirementParserOutput struct {
	ProjectIdentity    ProjectIdentity    `json:"project_identity"`
	DeploymentProfile  DeploymentProfile  `json:"deployment_profile"`
	ResourceRequirements ResourceRequirements `json:"resource_requirements"`
	Dependencies       []Dependency       `json:"dependencies"`
	ExposedEndpoints   []ExposedEndpoint  `json:"exposed_endpoints"`
	EnvironmentVariables []EnvVarDefinition `json:"environment_variables"`
	PotentialRisks     []string           `json:"potential_risks"`
}

type DeploymentProfile struct {
	Type            string `json:"type"`
	BuildRequired   bool   `json:"build_required"`
	Runtime         string `json:"runtime"`
	PackageManager  string `json:"package_manager,omitempty"`
	EstimatedComplexity string `json:"estimated_complexity"`
}

type ResourceRequirements struct {
	CPUCores  int `json:"cpu_cores"`
	MemoryGB  int `json:"memory_gb"`
	DiskGB    int `json:"disk_gb"`
	Network   string `json:"network"`
}

type Dependency struct {
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	Required bool     `json:"required"`
	Candidates []string `json:"candidates,omitempty"`
	Version  string   `json:"version,omitempty"`
}

type ExposedEndpoint struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Purpose  string `json:"purpose"`
}

type EnvVarDefinition struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Sensitive   bool   `json:"sensitive"`
	Description string `json:"description,omitempty"`
}

type ProjectIdentity struct {
	Name            string `json:"name"`
	PrimaryLanguage string `json:"primary_language"`
}

type CodeAnalyzerInput struct {
	RepoURL         string                  `json:"repo_url"`
	Branch          string                  `json:"branch,omitempty"`
	ProjectAnalysis *RequirementParserOutput `json:"project_analysis"`
}

type CodeAnalyzerResult struct {
	DeploySteps []DeployStep `json:"deploy_steps"`
}

type DeployStep struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
	Timeout     int      `json:"timeout_seconds"`
	Rollback    []string `json:"rollback,omitempty"`
	Verification string  `json:"verification,omitempty"`
}

type DeploymentExecutorInput struct {
	ServerID      string           `json:"server_id"`
	ProjectID     string           `json:"project_id"`
	CodeAnalysis  *CodeAnalyzerResult `json:"code_analysis"`
	DeploySteps   []DeployStep     `json:"deploy_steps"`
}

type DeploymentExecutorResult struct {
	Success      bool               `json:"success"`
	Steps        []StepExecutionResult `json:"steps"`
	Endpoints    []ServiceEndpoint  `json:"endpoints,omitempty"`
	ErrorMessage string             `json:"error_message,omitempty"`
	Errors       []DeployError      `json:"errors,omitempty"`
}

type StepExecutionResult struct {
	StepID      string `json:"step_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
	Duration    int64  `json:"duration_ms"`
	RetryCount  int    `json:"retry_count,omitempty"`
}

type DeployError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Severity  string `json:"severity"`
	Step      string `json:"step,omitempty"`
	Solution  string `json:"solution,omitempty"`
}

type TroubleshooterInput struct {
	ErrorLog    string                 `json:"error_log"`
	ExecContext *DeploymentExecutorResult `json:"exec_context"`
}

type TroubleshooterResult struct {
	Confidence      float64          `json:"confidence"`
	RootCause       string           `json:"root_cause"`
	RemediationPlan *RemediationPlan `json:"remediation_plan,omitempty"`
}

type RemediationPlan struct {
	Steps []RemediationStep `json:"steps"`
}

type RemediationStep struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type KnowledgeStorageInput struct {
	DeploymentID    string                  `json:"deployment_id"`
	Success         bool                    `json:"success"`
	ExecutionResult *DeploymentExecutorResult `json:"execution_result"`
	CodeAnalysis    *CodeAnalyzerResult     `json:"code_analysis"`
}
