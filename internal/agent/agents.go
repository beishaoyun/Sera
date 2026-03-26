package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/servermind/aixm/internal/ssh"
)

// Agent 接口定义
type Agent interface {
	// Name 返回 Agent 名称
	Name() string
	// Execute 执行 Agent 任务
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	// ValidateInput 验证输入
	ValidateInput(input interface{}) error
}

// AgentState Agent 状态
type AgentState string

const (
	AgentStateIdle       AgentState = "idle"
	AgentStateRunning    AgentState = "running"
	AgentStateCompleted  AgentState = "completed"
	AgentStateFailed     AgentState = "failed"
	AgentStateWaiting    AgentState = "waiting" // 等待外部输入
)

// AgentResult Agent 执行结果
type AgentResult struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	State     AgentState  `json:"state"`
	Confidence float64    `json:"confidence,omitempty"` // 置信度 0-1
}

// ============================================================================
// RequirementParser Agent - 需求解析器
// ============================================================================

// RequirementParserInput 输入结构
type RequirementParserInput struct {
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch,omitempty"`
	UserRequest string `json:"user_request,omitempty"` // 用户的额外说明
}

// RequirementParserOutput 输出结构
type RequirementParserOutput struct {
	ProjectIdentity    ProjectIdentity    `json:"project_identity"`
	DeploymentProfile  DeploymentProfile  `json:"deployment_profile"`
	ResourceRequirements ResourceRequirements `json:"resource_requirements"`
	Dependencies       []Dependency       `json:"dependencies"`
	ExposedEndpoints   []ExposedEndpoint  `json:"exposed_endpoints"`
	EnvironmentVariables []EnvVarDefinition `json:"environment_variables"`
	PotentialRisks     []string           `json:"potential_risks"`
}

type ProjectIdentity struct {
	Name           string `json:"name"`
	PrimaryLanguage string `json:"primary_language"`
	FrameWork      string `json:"framework,omitempty"`
	Architecture   string `json:"architecture,omitempty"`
}

type DeploymentProfile struct {
	Type            string `json:"type"`             // node_web_application, python_api, etc.
	BuildRequired   bool   `json:"build_required"`
	Runtime         string `json:"runtime"`
	PackageManager  string `json:"package_manager,omitempty"`
	EstimatedComplexity string `json:"estimated_complexity"` // simple, medium, complex
}

type ResourceRequirements struct {
	CPUCores  int `json:"cpu_cores"`
	MemoryGB  int `json:"memory_gb"`
	DiskGB    int `json:"disk_gb"`
	Network   string `json:"network"` // egress_only, ingress_required
}

type Dependency struct {
	Type     string   `json:"type"`             // service, database, cache, external_api
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

// RequirementParser RequirementParser Agent 实现
type RequirementParser struct {
	llmClient LLMClient
}

// NewRequirementParser 创建需求解析 Agent
func NewRequirementParser(llmClient LLMClient) *RequirementParser {
	return &RequirementParser{
		llmClient: llmClient,
	}
}

func (a *RequirementParser) Name() string {
	return "RequirementParser"
}

func (a *RequirementParser) ValidateInput(input interface{}) error {
	req, ok := input.(*RequirementParserInput)
	if !ok {
		return fmt.Errorf("invalid input type, expected *RequirementParserInput")
	}
	if req.RepoURL == "" {
		return fmt.Errorf("repo_url is required")
	}
	return nil
}

func (a *RequirementParser) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	req, ok := input.(*RequirementParserInput)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	logrus.WithFields(logrus.Fields{
		"agent":  a.Name(),
		"repo":   req.RepoURL,
		"branch": req.Branch,
	}).Info("Starting requirement analysis")

	// 1. 解析 GitHub URL
	repoInfo, err := parseGitHubURL(req.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}

	// 2. 获取仓库元数据（通过 GitHub API）
	metadata, err := fetchGitHubRepoMetadata(ctx, repoInfo.Owner, repoInfo.Repo)
	if err != nil {
		logrus.Warnf("Failed to fetch repo metadata: %v", err)
		// 继续执行，可能 README 中有足够信息
	}

	// 3. 获取文件树
	fileTree, err := fetchGitHubFileTree(ctx, repoInfo.Owner, repoInfo.Repo, req.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file tree: %w", err)
	}

	// 4. 获取 README 内容
	readmeContent, err := fetchGitHubREADME(ctx, repoInfo.Owner, repoInfo.Repo, req.Branch)
	if err != nil {
		logrus.Warnf("Failed to fetch README: %v", err)
	}

	// 5. 识别项目类型和关键文件
	projectAnalysis := analyzeProjectStructure(fileTree)

	// 6. 使用 LLM 分析 README 和项目结构，生成结构化输出
	prompt := buildRequirementParserPrompt(repoInfo, metadata, fileTree, readmeContent, projectAnalysis, req.UserRequest)

	llmResponse, err := a.llmClient.Generate(ctx, prompt, LLMOptions{
		Temperature: 0.3,
		MaxTokens:   2000,
		JSONSchema:  getRequirementParserSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// 7. 解析 LLM 输出
	var output RequirementParserOutput
	if err := json.Unmarshal([]byte(llmResponse), &output); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// 8. 填充缺失的默认值
	output = fillDefaultValues(output, projectAnalysis)

	logrus.WithFields(logrus.Fields{
		"agent":      a.Name(),
		"project":    output.ProjectIdentity.Name,
		"type":       output.DeploymentProfile.Type,
		"complexity": output.DeploymentProfile.EstimatedComplexity,
	}).Info("Requirement analysis completed")

	return &output, nil
}

// ============================================================================
// CodeAnalyzer Agent - 代码分析器
// ============================================================================

// CodeAnalyzerInput 输入结构
type CodeAnalyzerInput struct {
	RepoURL         string `json:"repo_url"`
	Branch          string `json:"branch,omitempty"`
	ProjectAnalysis *RequirementParserOutput `json:"project_analysis"`
}

// CodeAnalyzerOutput 输出结构
type CodeAnalyzerOutput struct {
	BuildConfig      BuildConfig      `json:"build_config"`
	RuntimeConfig    RuntimeConfig    `json:"runtime_config"`
	Dockerfile       *DockerfileConfig `json:"dockerfile,omitempty"`
	DockerCompose    *DockerComposeConfig `json:"docker_compose,omitempty"`
	ServiceTopology  []ServiceRelation `json:"service_topology"`
	HealthChecks     []HealthCheck    `json:"health_checks"`
	SecurityFindings []SecurityFinding `json:"security_findings"`
	DeploySteps      []DeployStep     `json:"deploy_steps"`
}

type BuildConfig struct {
	Steps       []BuildStep `json:"steps"`
	Artifacts   []string    `json:"artifacts"`
	BuildCmd    string      `json:"build_cmd,omitempty"`
	BuildOutput string      `json:"build_output,omitempty"`
}

type BuildStep struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Timeout     int    `json:"timeout_seconds"`
}

type RuntimeConfig struct {
	StartCmd      string            `json:"start_cmd"`
	WorkingDir    string            `json:"working_dir"`
	Ports         []int             `json:"ports"`
	EnvVars       map[string]string `json:"env_vars"`
	Volumes       []string          `json:"volumes"`
	RestartPolicy string            `json:"restart_policy"`
}

type DockerfileConfig struct {
	BaseImage    string   `json:"base_image"`
	BuildStages  []string `json:"build_stages"`
	RunCommands  []string `json:"run_commands"`
	ExposePorts  []int    `json:"expose_ports"`
	User         string   `json:"user"`
	Workdir      string   `json:"workdir"`
	Healthcheck  string   `json:"healthcheck"`
	FullContent  string   `json:"full_content"` // 完整的 Dockerfile 内容
}

type DockerComposeConfig struct {
	Services   map[string]ServiceConfig `json:"services"`
	Networks   []string                 `json:"networks,omitempty"`
	Volumes    []string                 `json:"volumes,omitempty"`
	FullContent string                  `json:"full_content"`
}

type ServiceConfig struct {
	Image       string            `json:"image,omitempty"`
	Build       string            `json:"build,omitempty"`
	Ports       []string          `json:"ports"`
	Environment map[string]string `json:"environment"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Volumes     []string          `json:"volumes,omitempty"`
}

type ServiceRelation struct {
	Service    string   `json:"service"`
	DependsOn  []string `json:"depends_on"`
	Provides   []string `json:"provides"`
	Ports      []int    `json:"ports"`
}

type HealthCheck struct {
	Type      string `json:"type"` // http, tcp, command
	Endpoint  string `json:"endpoint,omitempty"`
	Port      int    `json:"port"`
	Interval  int    `json:"interval_seconds"`
	Timeout   int    `json:"timeout_seconds"`
	Retries   int    `json:"retries"`
	Command   string `json:"command,omitempty"`
}

type SecurityFinding struct {
	Severity   string `json:"severity"` // critical, high, medium, low
	Type       string `json:"type"`     // secret, vulnerability, misconfiguration
	Path       string `json:"path"`
	Description string `json:"description"`
	Recommendation string `json:"recommendation"`
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

// CodeAnalyzer CodeAnalyzer Agent 实现
type CodeAnalyzer struct {
	llmClient LLMClient
}

// NewCodeAnalyzer 创建代码分析 Agent
func NewCodeAnalyzer(llmClient LLMClient) *CodeAnalyzer {
	return &CodeAnalyzer{
		llmClient: llmClient,
	}
}

func (a *CodeAnalyzer) Name() string {
	return "CodeAnalyzer"
}

func (a *CodeAnalyzer) ValidateInput(input interface{}) error {
	req, ok := input.(*CodeAnalyzerInput)
	if !ok {
		return fmt.Errorf("invalid input type, expected *CodeAnalyzerInput")
	}
	if req.RepoURL == "" {
		return fmt.Errorf("repo_url is required")
	}
	return nil
}

func (a *CodeAnalyzer) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	req, ok := input.(*CodeAnalyzerInput)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	logrus.WithFields(logrus.Fields{
		"agent": a.Name(),
		"repo":  req.RepoURL,
	}).Info("Starting code analysis")

	// 1. 获取完整代码仓库
	repoPath, err := cloneRepository(ctx, req.RepoURL, req.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	defer cleanupRepo(repoPath)

	// 2. 深度分析代码结构
	codeStructure, err := analyzeCodeStructure(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze code structure: %w", err)
	}

	// 3. 分析依赖
	dependencies, err := analyzeDependencies(repoPath, req.ProjectAnalysis)
	if err != nil {
		logrus.Warnf("Dependency analysis warning: %v", err)
	}

	// 4. 扫描安全问题
	securityFindings := scanSecurity(repoPath)

	// 5. 使用 LLM 生成部署配置
	prompt := buildCodeAnalyzerPrompt(codeStructure, dependencies, req.ProjectAnalysis)

	llmResponse, err := a.llmClient.Generate(ctx, prompt, LLMOptions{
		Temperature: 0.2,
		MaxTokens:   3000,
		JSONSchema:  getCodeAnalyzerSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	var output CodeAnalyzerOutput
	if err := json.Unmarshal([]byte(llmResponse), &output); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// 添加安全发现
	output.SecurityFindings = securityFindings

	logrus.WithFields(logrus.Fields{
		"agent":     a.Name(),
		"hasDocker": output.Dockerfile != nil,
		"steps":     len(output.DeploySteps),
	}).Info("Code analysis completed")

	return &output, nil
}

// ============================================================================
// DeploymentExecutor Agent - 部署执行器
// ============================================================================

// DeploymentExecutorInput 输入结构
type DeploymentExecutorInput struct {
	ServerID       string             `json:"server_id"`
	ProjectID      string             `json:"project_id"`
	CodeAnalysis   *CodeAnalyzerOutput `json:"code_analysis"`
	DeploySteps    []DeployStep       `json:"deploy_steps"`
}

// DeploymentExecutorOutput 输出结构
type DeploymentExecutorOutput struct {
	Success   bool                `json:"success"`
	Steps     []StepExecutionResult `json:"steps"`
	Endpoints []ServiceEndpoint   `json:"endpoints,omitempty"`
	Errors    []DeployError       `json:"errors,omitempty"`
	Duration  int64               `json:"duration_ms"`
}

type StepExecutionResult struct {
	StepID      string `json:"step_id"`
	Name        string `json:"name"`
	Status      string `json:"status"` // success, failed, skipped
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
	Duration    int64  `json:"duration_ms"`
	RetryCount  int    `json:"retry_count,omitempty"`
}

// DeploymentExecutor DeploymentExecutor Agent 实现
type DeploymentExecutor struct {
	sshClientFactory SSHClientFactory
}

// NewDeploymentExecutor 创建部署执行 Agent
func NewDeploymentExecutor(sshFactory SSHClientFactory) *DeploymentExecutor {
	return &DeploymentExecutor{
		sshClientFactory: sshFactory,
	}
}

func (a *DeploymentExecutor) Name() string {
	return "DeploymentExecutor"
}

func (a *DeploymentExecutor) ValidateInput(input interface{}) error {
	req, ok := input.(*DeploymentExecutorInput)
	if !ok {
		return fmt.Errorf("invalid input type, expected *DeploymentExecutorInput")
	}
	if req.ServerID == "" {
		return fmt.Errorf("server_id is required")
	}
	return nil
}

func (a *DeploymentExecutor) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	req, ok := input.(*DeploymentExecutorInput)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	logrus.WithFields(logrus.Fields{
		"agent":    a.Name(),
		"server":   req.ServerID,
		"steps":    len(req.DeploySteps),
	}).Info("Starting deployment execution")

	output := &DeploymentExecutorOutput{
		Success: true,
		Steps:   make([]StepExecutionResult, 0, len(req.DeploySteps)),
	}

	startTime := time.Now()

	// 获取 SSH 连接
	sshClient, err := a.sshClientFactory.GetClient(ctx, req.ServerID)
	if err != nil {
		output.Success = false
		output.Errors = append(output.Errors, DeployError{
			Code:    "SSH_CONNECT_FAILED",
			Message: err.Error(),
		})
		return output, nil
	}
	defer sshClient.Close()

	// 执行每个部署步骤
	for _, step := range req.DeploySteps {
		select {
		case <-ctx.Done():
			output.Success = false
			output.Errors = append(output.Errors, DeployError{
				Code:    "CANCELLED",
				Message: "Deployment cancelled by user",
			})
			return output, nil
		default:
		}

		stepResult := a.executeStep(ctx, sshClient, step)
		output.Steps = append(output.Steps, stepResult)

		if stepResult.Status == "failed" {
			output.Success = false
			output.Errors = append(output.Errors, DeployError{
				Code:    "STEP_FAILED",
				Message: stepResult.Error,
				Step:    step.ID,
			})

			// 执行回滚
			if len(step.Rollback) > 0 {
				logrus.Infof("Executing rollback for step %s", step.ID)
				a.executeRollback(ctx, sshClient, step)
			}

			break
		}
	}

	output.Duration = time.Since(startTime).Milliseconds()

	logrus.WithFields(logrus.Fields{
		"agent":    a.Name(),
		"success":  output.Success,
		"duration": output.Duration,
	}).Info("Deployment execution completed")

	return output, nil
}

func (a *DeploymentExecutor) executeStep(ctx context.Context, sshClient ssh.SSHClient, step DeployStep) StepExecutionResult {
	result := StepExecutionResult{
		StepID: step.ID,
		Name:   step.Name,
	}

	startTime := time.Now()

	// 执行命令
	var allOutput []string
	for _, cmd := range step.Commands {
		execResult, err := sshClient.Execute(ctx, cmd, ssh.ExecuteOptions{
			Timeout: time.Duration(step.Timeout) * time.Second,
		})

		if execResult != nil {
			allOutput = append(allOutput, execResult.Output)
		}

		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}
	}

	// 执行验证
	if step.Verification != "" {
		_, err := sshClient.Execute(ctx, step.Verification, ssh.ExecuteOptions{})
		if err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("verification failed: %v", err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		}
	}

	result.Status = "success"
	result.Output = strings.Join(allOutput, "\n")
	result.Duration = time.Since(startTime).Milliseconds()
	return result
}

func (a *DeploymentExecutor) executeRollback(ctx context.Context, sshClient ssh.SSHClient, step DeployStep) {
	for _, cmd := range step.Rollback {
		_, _ = sshClient.Execute(ctx, cmd, ssh.ExecuteOptions{
			Timeout: 60 * time.Second,
		})
	}
}

// ============================================================================
// Troubleshooter Agent - 故障诊断器
// ============================================================================

// TroubleshooterInput 输入结构
type TroubleshooterInput struct {
	ErrorLog      string                 `json:"error_log"`
	ExecContext   *DeploymentExecutorInput `json:"exec_context"`
	StepResults   []StepExecutionResult  `json:"step_results"`
	KnowledgeCases []KnowledgeCase       `json:"knowledge_cases,omitempty"`
}

// TroubleshooterOutput 输出结构
type TroubleshooterOutput struct {
	 diagnoses      []Diagnosis          `json:"diagnoses"`
	RootCause       string               `json:"root_cause,omitempty"`
	Confidence      float64              `json:"confidence"`
	RemediationPlan *RemediationPlan     `json:"remediation_plan,omitempty"`
	NeedHumanHelp   bool                 `json:"need_human_help"`
}

type Diagnosis struct {
	Category    string  `json:"category"` // build_error, dependency_error, runtime_error, etc.
	Description string  `json:"description"`
	Evidence    []string `json:"evidence"`
	Severity    string  `json:"severity"`
}

type RemediationPlan struct {
	Steps       []RemediationStep `json:"steps"`
	EstimatedTime int             `json:"estimated_time_seconds"`
	RiskLevel   string            `json:"risk_level"` // low, medium, high
	RequiresApproval bool         `json:"requires_approval"`
}

type RemediationStep struct {
	ID          string `json:"id"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Risk        string `json:"risk"`
	Rollback    string `json:"rollback,omitempty"`
}

// Troubleshooter Troubleshooter Agent 实现
type Troubleshooter struct {
	llmClient    LLMClient
	knowledgeRepo KnowledgeRepository
}

// NewTroubleshooter 创建故障诊断 Agent
func NewTroubleshooter(llmClient LLMClient, knowledgeRepo KnowledgeRepository) *Troubleshooter {
	return &Troubleshooter{
		llmClient:    llmClient,
		knowledgeRepo: knowledgeRepo,
	}
}

func (a *Troubleshooter) Name() string {
	return "Troubleshooter"
}

func (a *Troubleshooter) ValidateInput(input interface{}) error {
	req, ok := input.(*TroubleshooterInput)
	if !ok {
		return fmt.Errorf("invalid input type, expected *TroubleshooterInput")
	}
	if req.ErrorLog == "" {
		return fmt.Errorf("error_log is required")
	}
	return nil
}

func (a *Troubleshooter) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	req, ok := input.(*TroubleshooterInput)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	logrus.WithFields(logrus.Fields{
		"agent":     a.Name(),
		"errorLogs": len(req.ErrorLog),
	}).Info("Starting troubleshooting")

	// 1. 错误分类
	errorCategory := classifyError(req.ErrorLog)

	// 2. 从知识库检索相似案例
	similarCases, err := a.knowledgeRepo.SearchSimilar(ctx, req.ErrorLog, errorCategory, 5)
	if err != nil {
		logrus.Warnf("Knowledge search failed: %v", err)
	}

	// 3. 根因分析
	prompt := buildTroubleshooterPrompt(req, errorCategory, similarCases)

	llmResponse, err := a.llmClient.Generate(ctx, prompt, LLMOptions{
		Temperature: 0.1,
		MaxTokens:   2500,
		JSONSchema:  getTroubleshooterSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	var output TroubleshooterOutput
	if err := json.Unmarshal([]byte(llmResponse), &output); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// 4. 判断是否需要人工帮助
	if output.Confidence < 0.5 {
		output.NeedHumanHelp = true
	}

	logrus.WithFields(logrus.Fields{
		"agent":       a.Name(),
		"rootCause":   output.RootCause,
		"confidence":  output.Confidence,
		"needHuman":   output.NeedHumanHelp,
	}).Info("Troubleshooting completed")

	return &output, nil
}

// ============================================================================
// 辅助接口和类型
// ============================================================================

// LLMClient LLM 客户端接口
type LLMClient interface {
	Generate(ctx context.Context, prompt string, opts LLMOptions) (string, error)
}

type LLMOptions struct {
	Temperature float64
	MaxTokens   int
	JSONSchema  map[string]interface{}
}

// SSHClientFactory SSH 客户端工厂接口
type SSHClientFactory interface {
	GetClient(ctx context.Context, serverID string) (ssh.SSHClient, error)
}

// KnowledgeRepository 知识库仓储接口
type KnowledgeRepository interface {
	SearchSimilar(ctx context.Context, query, category string, limit int) ([]KnowledgeCase, error)
}

// 以下函数需要在实际实现中补充
func parseGitHubURL(url string) (*RepoInfo, error) { return nil, nil }
func fetchGitHubRepoMetadata(ctx context.Context, owner, repo string) (*RepoMetadata, error) { return nil, nil }
func fetchGitHubFileTree(ctx context.Context, owner, repo, branch string) ([]FileEntry, error) { return nil, nil }
func fetchGitHubREADME(ctx context.Context, owner, repo, branch string) (string, error) { return "", nil }
func analyzeProjectStructure(fileTree []FileEntry) *ProjectAnalysis { return nil }
func buildRequirementParserPrompt(repo *RepoInfo, meta *RepoMetadata, tree []FileEntry, readme, analysis interface{}, userReq string) string { return "" }
func getRequirementParserSchema() map[string]interface{} { return nil }
func fillDefaultValues(output RequirementParserOutput, analysis *ProjectAnalysis) RequirementParserOutput { return output }
func cloneRepository(ctx context.Context, url, branch string) (string, error) { return "", nil }
func cleanupRepo(path string) {}
func analyzeCodeStructure(path string) (*CodeStructure, error) { return nil, nil }
func analyzeDependencies(path string, analysis *RequirementParserOutput) (*Dependencies, error) { return nil, nil }
func scanSecurity(path string) []SecurityFinding { return nil }
func buildCodeAnalyzerPrompt(structure, deps, analysis interface{}) string { return "" }
func getCodeAnalyzerSchema() map[string]interface{} { return nil }
func classifyError(log string) string { return "" }
func buildTroubleshooterPrompt(req *TroubleshooterInput, category string, cases []KnowledgeCase) string { return "" }
func getTroubleshooterSchema() map[string]interface{} { return nil }

// 辅助类型
type RepoInfo struct {
	Owner string
	Repo  string
	Branch string
}
type RepoMetadata struct {
	Stars int `json:"stars"`
	Language string `json:"language"`
	UpdatedAt string `json:"updated_at"`
}
type FileEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}
type ProjectAnalysis struct{}
type CodeStructure struct{}
type Dependencies struct{}
