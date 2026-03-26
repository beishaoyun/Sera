package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// LLMProvider LLM 提供商类型
type LLMProvider string

const (
	ProviderOpenAI  LLMProvider = "openai"
	ProviderClaude  LLMProvider = "claude"
	ProviderOllama  LLMProvider = "ollama"
)

// LLMClient LLM 客户端接口
type LLMClient interface {
	// Generate 生成文本响应
	Generate(ctx context.Context, prompt string, opts GenerateOptions) (string, error)
	// GenerateJSON 生成 JSON 格式响应
	GenerateJSON(ctx context.Context, prompt string, schema map[string]interface{}, opts GenerateOptions) (interface{}, error)
	// Chat 多轮对话
	Chat(ctx context.Context, messages []Message, opts GenerateOptions) (string, error)
	// AnalyzeProject 分析项目
	AnalyzeProject(ctx context.Context, repoURL, readmeContent, fileTree string) (*ProjectAnalysis, error)
	// ParseTutorial 解析教程
	ParseTutorial(ctx context.Context, tutorialContent string) (*TutorialSteps, error)
	// GenerateTroubleshooting 生成故障诊断
	GenerateTroubleshooting(ctx context.Context, errorLog, context string) (*TroubleshootingResult, error)
}

// GenerateOptions 生成选项
type GenerateOptions struct {
	Temperature float64
	MaxTokens   int
	Model       string
}

// Message 对话消息
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ProjectAnalysis 项目分析结果
type ProjectAnalysis struct {
	Name           string            `json:"name"`
	Language       string            `json:"language"`
	FrameWork      string            `json:"framework"`
	DeployType     string            `json:"deploy_type"`
	HasDockerfile  bool              `json:"has_dockerfile"`
	BuildSteps     []string          `json:"build_steps"`
	RunCommand     string            `json:"run_command"`
	Ports          []int             `json:"ports"`
	EnvVars        []EnvVar          `json:"env_vars"`
	Dependencies   []string          `json:"dependencies"`
	ResourceReqs   ResourceRequirements `json:"resource_requirements"`
	PotentialRisks []string          `json:"potential_risks"`
}

// TutorialSteps 教程步骤
type TutorialSteps struct {
	Title       string        `json:"title"`
	Source      string        `json:"source"` // 来源：github, csdn, jianshu, etc.
	Steps       []DeployStep  `json:"steps"`
	Prereqs     []string      `json:"prerequisites"`
	Commands    []string      `json:"commands"`
	ConfigFiles []ConfigFile  `json:"config_files"`
	Warnings    []string      `json:"warnings"`
}

// DeployStep 部署步骤
type DeployStep struct {
	Order       int    `json:"order"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Manual      bool   `json:"manual"` // 是否需要手动操作
}

// ConfigFile 配置文件
type ConfigFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Purpose string `json:"purpose"`
}

// TroubleshootingResult 故障诊断结果
type TroubleshootingResult struct {
	ErrorType     string   `json:"error_type"`
	RootCause     string   `json:"root_cause"`
	Confidence    float64  `json:"confidence"`
	Solutions     []Solution `json:"solutions"`
	NeedHumanHelp bool     `json:"need_human_help"`
}

// Solution 解决方案
type Solution struct {
	Order       int      `json:"order"`
	Title       string   `json:"title"`
	Commands    []string `json:"commands"`
	Description string   `json:"description"`
	Risk        string   `json:"risk"` // low, medium, high
}

// EnvVar 环境变量
type EnvVar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Sensitive   bool   `json:"sensitive"`
}

// ResourceRequirements 资源需求
type ResourceRequirements struct {
	CPUCores   int `json:"cpu_cores"`
	MemoryMB   int `json:"memory_mb"`
	DiskGB     int `json:"disk_gb"`
}

// ============================================================================
// OpenAI 客户端实现
// ============================================================================

// OpenAIClient OpenAI API 客户端
type OpenAIClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAIClient 创建 OpenAI 客户端
func NewOpenAIClient(apiKey, baseURL, model string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4-turbo-preview"
	}
	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *OpenAIClient) Generate(ctx context.Context, prompt string, opts GenerateOptions) (string, error) {
	messages := []Message{
		{Role: "system", Content: "你是一个专业的 DevOps 工程师，擅长分析项目和部署服务。"},
		{Role: "user", Content: prompt},
	}
	return c.Chat(ctx, messages, opts)
}

func (c *OpenAIClient) GenerateJSON(ctx context.Context, prompt string, schema map[string]interface{}, opts GenerateOptions) (interface{}, error) {
	systemPrompt := "你是一个专业的 API，只返回有效的 JSON 格式响应，不包含任何其他解释。"
	if schema != nil {
		schemaJSON, _ := json.Marshal(schema)
		systemPrompt = fmt.Sprintf("%s\n\n请严格按照以下 JSON Schema 返回:\n%s", systemPrompt, string(schemaJSON))
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	response, err := c.Chat(ctx, messages, opts)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return result, nil
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []Message, opts GenerateOptions) (string, error) {
	if opts.Model == "" {
		opts.Model = c.model
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 4000
	}
	if opts.Temperature == 0 {
		opts.Temperature = 0.3
	}

	reqBody := map[string]interface{}{
		"model":       opts.Model,
		"messages":    messages,
		"temperature": opts.Temperature,
		"max_tokens":  opts.MaxTokens,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(reqJSON))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return apiResp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) AnalyzeProject(ctx context.Context, repoURL, readmeContent, fileTree string) (*ProjectAnalysis, error) {
	prompt := fmt.Sprintf(`请分析以下 GitHub 项目并生成部署配置：

项目地址：%s

文件结构:
%s

README 内容:
%s

请以 JSON 格式返回分析结果，包含:
- name: 项目名称
- language: 主要编程语言
- framework: 框架名称
- deploy_type: 部署类型 (docker/native/k8s)
- has_dockerfile: 是否有 Dockerfile
- build_steps: 构建步骤数组
- run_command: 运行命令
- ports: 端口列表
- env_vars: 环境变量列表 (name, description, required)
- dependencies: 依赖列表
- resource_requirements: 资源需求 (cpu_cores, memory_mb, disk_gb)
- potential_risks: 潜在风险列表`, repoURL, fileTree, readmeContent)

	response, err := c.GenerateJSON(ctx, prompt, getProjectAnalysisSchema(), GenerateOptions{})
	if err != nil {
		return nil, err
	}

	var analysis ProjectAnalysis
	data, _ := json.Marshal(response)
	json.Unmarshal(data, &analysis)

	logrus.WithFields(logrus.Fields{
		"project":   analysis.Name,
		"language":  analysis.Language,
		"framework": analysis.FrameWork,
	}).Info("Project analyzed")

	return &analysis, nil
}

func (c *OpenAIClient) ParseTutorial(ctx context.Context, tutorialContent string) (*TutorialSteps, error) {
	prompt := fmt.Sprintf(`请分析以下部署教程并提取步骤：

教程内容:
%s

请提取:
- title: 教程标题
- source: 来源 (github/csdn/jianshu/zhihu/other)
- steps: 部署步骤列表 (order, title, description, command, manual)
- prerequisites: 前置条件列表
- commands: 所有命令汇总
- config_files: 配置文件示例
- warnings: 注意事项`, tutorialContent)

	response, err := c.GenerateJSON(ctx, prompt, getTutorialSchema(), GenerateOptions{})
	if err != nil {
		return nil, err
	}

	var steps TutorialSteps
	data, _ := json.Marshal(response)
	json.Unmarshal(data, &steps)

	return &steps, nil
}

func (c *OpenAIClient) GenerateTroubleshooting(ctx context.Context, errorLog, context string) (*TroubleshootingResult, error) {
	prompt := fmt.Sprintf(`请诊断以下部署错误：

错误日志:
%s

上下文:
%s

请分析:
- error_type: 错误类型
- root_cause: 根本原因
- confidence: 置信度 (0-1)
- solutions: 解决方案列表 (order, title, commands, description, risk)
- need_human_help: 是否需要人工帮助`, errorLog, context)

	response, err := c.GenerateJSON(ctx, prompt, getTroubleshootingSchema(), GenerateOptions{
		Temperature: 0.1, // 低温保证诊断准确性
	})
	if err != nil {
		return nil, err
	}

	var result TroubleshootingResult
	data, _ := json.Marshal(response)
	json.Unmarshal(data, &result)

	return &result, nil
}

// Schema 函数
func getProjectAnalysisSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]string{"type": "string"},
			"language": map[string]string{"type": "string"},
			"framework": map[string]string{"type": "string"},
			"deploy_type": map[string]string{"type": "string", "enum": []string{"docker", "native", "k8s"}},
			"has_dockerfile": map[string]bool{"type": "boolean"},
			"build_steps": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			"run_command": map[string]string{"type": "string"},
			"ports": map[string]interface{}{"type": "array", "items": map[string]int{"type": "integer"}},
			"env_vars": map[string]interface{}{"type": "array", "items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]string{"type": "string"},
					"description": map[string]string{"type": "string"},
					"required": map[string]bool{"type": "boolean"},
					"sensitive": map[string]bool{"type": "boolean"},
				},
			}},
			"dependencies": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			"resource_requirements": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cpu_cores": map[string]int{"type": "integer"},
					"memory_mb": map[string]int{"type": "integer"},
					"disk_gb": map[string]int{"type": "integer"},
				},
			},
			"potential_risks": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
		},
		"required": []string{"name", "language", "deploy_type", "build_steps", "run_command"},
	}
}

func getTutorialSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]string{"type": "string"},
			"source": map[string]string{"type": "string"},
			"steps": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order": map[string]int{"type": "integer"},
						"title": map[string]string{"type": "string"},
						"description": map[string]string{"type": "string"},
						"command": map[string]string{"type": "string"},
						"manual": map[string]bool{"type": "boolean"},
					},
				},
			},
			"prerequisites": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			"commands": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
			"config_files": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]string{"type": "string"},
						"content": map[string]string{"type": "string"},
						"purpose": map[string]string{"type": "string"},
					},
				},
			},
			"warnings": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
		},
		"required": []string{"title", "steps"},
	}
}

func getTroubleshootingSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"error_type": map[string]string{"type": "string"},
			"root_cause": map[string]string{"type": "string"},
			"confidence": map[string]float64{"type": "number"},
			"solutions": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order": map[string]int{"type": "integer"},
						"title": map[string]string{"type": "string"},
						"commands": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
						"description": map[string]string{"type": "string"},
						"risk": map[string]string{"type": "string"},
					},
				},
			},
			"need_human_help": map[string]bool{"type": "boolean"},
		},
		"required": []string{"error_type", "root_cause", "confidence", "solutions"},
	}
}

// ============================================================================
// Claude API 客户端实现
// ============================================================================

// ClaudeClient Anthropic Claude API 客户端
type ClaudeClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClaudeClient 创建 Claude 客户端
func NewClaudeClient(apiKey, model string) *ClaudeClient {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &ClaudeClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *ClaudeClient) Generate(ctx context.Context, prompt string, opts GenerateOptions) (string, error) {
	return c.Chat(ctx, []Message{{Role: "user", Content: prompt}}, opts)
}

func (c *ClaudeClient) GenerateJSON(ctx context.Context, prompt string, schema map[string]interface{}, opts GenerateOptions) (interface{}, error) {
	systemPrompt := "你是一个专业的 API，只返回有效的 JSON 格式响应，不包含任何其他解释。"
	response, err := c.Chat(ctx, []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("%s\n\n请返回 JSON 格式响应", prompt)},
	}, opts)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return result, nil
}

func (c *ClaudeClient) Chat(ctx context.Context, messages []Message, opts GenerateOptions) (string, error) {
	if opts.Model == "" {
		opts.Model = c.model
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 4000
	}
	if opts.Temperature == 0 {
		opts.Temperature = 0.3
	}

	// 分离 system message
	var systemMsg string
	var chatMessages []map[string]string
	for _, m := range messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			chatMessages = append(chatMessages, map[string]string{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}

	reqBody := map[string]interface{}{
		"model":       opts.Model,
		"messages":    chatMessages,
		"temperature": opts.Temperature,
		"max_tokens":  opts.MaxTokens,
	}
	if systemMsg != "" {
		reqBody["system"] = systemMsg
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqJSON))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
			Type string `json:"type"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return apiResp.Content[0].Text, nil
}

// 实现 LLMClient 接口的其他方法（委托给 Chat）
func (c *ClaudeClient) AnalyzeProject(ctx context.Context, repoURL, readmeContent, fileTree string) (*ProjectAnalysis, error) {
	// 与 OpenAI 类似的实现
	prompt := fmt.Sprintf(`请分析以下 GitHub 项目并生成部署配置：

项目地址：%s

文件结构:
%s

README 内容:
%s

请返回 JSON 格式的分析结果。`, repoURL, fileTree, readmeContent)

	response, err := c.GenerateJSON(ctx, prompt, getProjectAnalysisSchema(), GenerateOptions{})
	if err != nil {
		return nil, err
	}

	var analysis ProjectAnalysis
	data, _ := json.Marshal(response)
	json.Unmarshal(data, &analysis)

	return &analysis, nil
}

func (c *ClaudeClient) ParseTutorial(ctx context.Context, tutorialContent string) (*TutorialSteps, error) {
	prompt := fmt.Sprintf(`请分析以下部署教程并提取步骤：

教程内容:
%s

请返回 JSON 格式的教程步骤。`, tutorialContent)

	response, err := c.GenerateJSON(ctx, prompt, getTutorialSchema(), GenerateOptions{})
	if err != nil {
		return nil, err
	}

	var steps TutorialSteps
	data, _ := json.Marshal(response)
	json.Unmarshal(data, &steps)

	return &steps, nil
}

func (c *ClaudeClient) GenerateTroubleshooting(ctx context.Context, errorLog, context string) (*TroubleshootingResult, error) {
	prompt := fmt.Sprintf(`请诊断以下部署错误：

错误日志:
%s
上下文:
%s

请返回 JSON 格式的诊断结果。`, errorLog, context)

	response, err := c.GenerateJSON(ctx, prompt, getTroubleshootingSchema(), GenerateOptions{
		Temperature: 0.1,
	})
	if err != nil {
		return nil, err
	}

	var result TroubleshootingResult
	data, _ := json.Marshal(response)
	json.Unmarshal(data, &result)

	return &result, nil
}
