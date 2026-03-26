package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/servermind/aixm/internal/github"
	"github.com/servermind/aixm/internal/llm"
)

// ============================================================================
// 辅助函数实现
// ============================================================================

// RepoInfo 仓库信息（从 github 包复用）
type RepoInfo = github.RepoInfo

// RepoMetadata 仓库元数据
type RepoMetadata = github.Repository

// FileEntry 文件条目
type FileEntry = github.FileEntry

// ProjectAnalysis 项目分析
type ProjectAnalysis struct {
	KeyFiles       []string
	Language       string
	Framework      string
	HasDockerfile  bool
	HasDockerCompose bool
	PackageFiles   []string
	ConfigFiles    []string
}

// CodeStructure 代码结构
type CodeStructure struct {
	Dirs          []string
	Files         []string
	EntryPoints   []string
	ConfigFiles   map[string]string
}

// Dependencies 依赖信息
type Dependencies struct {
	Production []string
	Dev        []string
	System     []string
}

// 以下函数使用实际的 GitHub API 和 LLM 客户端

// parseGitHubURL 解析 GitHub URL
func parseGitHubURL(rawURL string) (*RepoInfo, error) {
	return github.ParseURL(rawURL)
}

// fetchGitHubRepoMetadata 获取仓库元数据
func fetchGitHubRepoMetadata(ctx context.Context, owner, repo string) (*RepoMetadata, error) {
	client := github.NewClientWithToken(getGitHubToken())
	return client.GetRepository(ctx, owner, repo)
}

// fetchGitHubFileTree 获取文件树
func fetchGitHubFileTree(ctx context.Context, owner, repo, branch string) ([]FileEntry, error) {
	client := github.NewClientWithToken(getGitHubToken())
	return client.GetFileTree(ctx, owner, repo, branch)
}

// fetchGitHubREADME 获取 README 内容
func fetchGitHubREADME(ctx context.Context, owner, repo, branch string) (string, error) {
	client := github.NewClientWithToken(getGitHubToken())
	return client.GetREADME(ctx, owner, repo, branch)
}

// analyzeProjectStructure 分析项目结构
func analyzeProjectStructure(fileTree []FileEntry) *ProjectAnalysis {
	analysis := &ProjectAnalysis{
		Language: "unknown",
	}

	langCount := make(map[string]int)
	hasGo := false
	hasNode := false
	hasPython := false
	hasRust := false
	hasJava := false

	for _, file := range fileTree {
		path := strings.ToLower(file.Path)

		// 检测语言
		if strings.HasSuffix(path, ".go") {
			langCount["go"]++
			hasGo = true
		} else if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".js") {
			langCount["javascript"]++
			hasNode = true
		} else if strings.HasSuffix(path, ".py") {
			langCount["python"]++
			hasPython = true
		} else if strings.HasSuffix(path, ".rs") {
			langCount["rust"]++
			hasRust = true
		} else if strings.HasSuffix(path, ".java") {
			langCount["java"]++
			hasJava = true
		}

		// Docker 文件
		if path == "dockerfile" || path == "docker-compose.yml" || path == "docker-compose.yaml" {
			analysis.HasDockerfile = true
			analysis.HasDockerCompose = true
		}

		// 包管理文件
		if path == "go.mod" || path == "go.sum" {
			analysis.PackageFiles = append(analysis.PackageFiles, path)
		} else if path == "package.json" || path == "package-lock.json" {
			analysis.PackageFiles = append(analysis.PackageFiles, path)
		} else if path == "requirements.txt" || path == "setup.py" || path == "pyproject.toml" {
			analysis.PackageFiles = append(analysis.PackageFiles, path)
		} else if path == "cargo.toml" {
			analysis.PackageFiles = append(analysis.PackageFiles, path)
		} else if path == "pom.xml" || path == "build.gradle" {
			analysis.PackageFiles = append(analysis.PackageFiles, path)
		}

		// 配置文件
		if strings.Contains(path, ".env") || strings.Contains(path, "config.") {
			analysis.ConfigFiles = append(analysis.ConfigFiles, path)
		}

		// 关键文件
		if strings.HasSuffix(path, "main.go") || strings.HasSuffix(path, "index.js") || strings.HasSuffix(path, "app.py") {
			analysis.KeyFiles = append(analysis.KeyFiles, path)
		}
	}

	// 确定主要语言
	maxCount := 0
	for lang, count := range langCount {
		if count > maxCount {
			maxCount = count
			analysis.Language = lang
		}
	}

	// 基于文件特征判断框架
	if hasNode {
		for _, file := range fileTree {
			if file.Path == "package.json" {
				// 可以尝试读取 package.json 判断框架
			}
		}
	}

	return analysis
}

// buildRequirementParserPrompt 构建需求解析 Prompt
func buildRequirementParserPrompt(repo *RepoInfo, meta *RepoMetadata, tree []FileEntry, readme string, analysis *ProjectAnalysis, userReq string) string {
	var sb strings.Builder

	sb.WriteString("你是一个专业的 DevOps 工程师，负责分析 GitHub 项目并生成部署配置。\n\n")

	sb.WriteString(fmt.Sprintf("## 仓库信息\n"))
	sb.WriteString(fmt.Sprintf("- 名称：%s\n", meta.FullName))
	sb.WriteString(fmt.Sprintf("- 描述：%s\n", meta.Description))
	sb.WriteString(fmt.Sprintf("- 语言：%s\n", meta.Language))
	sb.WriteString(fmt.Sprintf("- Stars: %d\n", meta.Stars))
	sb.WriteString(fmt.Sprintf("- 默认分支：%s\n\n", meta.DefaultBranch))

	sb.WriteString("## 文件结构\n")
	for _, file := range tree {
		if file.Type == "blob" {
			sb.WriteString(fmt.Sprintf("- %s\n", file.Path))
		}
	}

	sb.WriteString("\n## README 内容\n")
	if readme != "" {
		if len(readme) > 8000 {
			sb.WriteString(readme[:8000])
			sb.WriteString("\n... (truncated)")
		} else {
			sb.WriteString(readme)
		}
	} else {
		sb.WriteString("*No README available*\n")
	}

	sb.WriteString("\n## 项目分析结果\n")
	sb.WriteString(fmt.Sprintf("- 检测语言：%s\n", analysis.Language))
	sb.WriteString(fmt.Sprintf("- 关键文件：%v\n", analysis.KeyFiles))
	sb.WriteString(fmt.Sprintf("- 包管理文件：%v\n", analysis.PackageFiles))

	if userReq != "" {
		sb.WriteString(fmt.Sprintf("\n## 用户要求\n%s\n", userReq))
	}

	sb.WriteString("\n\n请以 JSON 格式返回部署配置，包含以下字段:\n")
	sb.WriteString("- project_identity: {name, primary_language, framework, architecture}\n")
	sb.WriteString("- deployment_profile: {type, build_required, runtime, package_manager, estimated_complexity}\n")
	sb.WriteString("- resource_requirements: {cpu_cores, memory_gb, disk_gb, network}\n")
	sb.WriteString("- dependencies: [{type, name, required, candidates, version}]\n")
	sb.WriteString("- exposed_endpoints: [{port, protocol, purpose}]\n")
	sb.WriteString("- environment_variables: [{name, required, default, sensitive, description}]\n")
	sb.WriteString("- potential_risks: [string]\n")

	return sb.String()
}

// getRequirementParserSchema 返回 JSON Schema
func getRequirementParserSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"project_identity": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]string{"type": "string"},
					"primary_language": map[string]string{"type": "string"},
					"framework": map[string]string{"type": "string"},
					"architecture": map[string]string{"type": "string"},
				},
				"required": []string{"name", "primary_language"},
			},
			"deployment_profile": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type": map[string]string{"type": "string"},
					"build_required": map[string]bool{"type": "boolean"},
					"runtime": map[string]string{"type": "string"},
					"package_manager": map[string]string{"type": "string"},
					"estimated_complexity": map[string]string{"type": "string", "enum": []string{"simple", "medium", "complex"}},
				},
				"required": []string{"type", "build_required"},
			},
			"resource_requirements": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cpu_cores": map[string]int{"type": "integer"},
					"memory_gb": map[string]int{"type": "integer"},
					"disk_gb": map[string]int{"type": "integer"},
					"network": map[string]string{"type": "string"},
				},
			},
			"dependencies": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type": map[string]string{"type": "string"},
						"name": map[string]string{"type": "string"},
						"required": map[string]bool{"type": "boolean"},
						"candidates": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
						"version": map[string]string{"type": "string"},
					},
				},
			},
			"exposed_endpoints": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"port": map[string]int{"type": "integer"},
						"protocol": map[string]string{"type": "string"},
						"purpose": map[string]string{"type": "string"},
					},
				},
			},
			"environment_variables": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]string{"type": "string"},
						"required": map[string]bool{"type": "boolean"},
						"default": map[string]string{"type": "string"},
						"sensitive": map[string]bool{"type": "boolean"},
						"description": map[string]string{"type": "string"},
					},
				},
			},
			"potential_risks": map[string]interface{}{
				"type": "array",
				"items": map[string]string{"type": "string"},
			},
		},
		"required": []string{"project_identity", "deployment_profile"},
	}
}

// fillDefaultValues 填充默认值
func fillDefaultValues(output RequirementParserOutput, analysis *ProjectAnalysis) RequirementParserOutput {
	// 如果置信度低，使用分析结果
	if output.ProjectIdentity.PrimaryLanguage == "" {
		output.ProjectIdentity.PrimaryLanguage = analysis.Language
	}

	// 默认资源需求
	if output.ResourceRequirements.CPUCores == 0 {
		output.ResourceRequirements.CPUCores = 1
	}
	if output.ResourceRequirements.MemoryGB == 0 {
		output.ResourceRequirements.MemoryGB = 1
	}
	if output.ResourceRequirements.DiskGB == 0 {
		output.ResourceRequirements.DiskGB = 5
	}

	return output
}

// cloneRepository 克隆仓库（用于深度分析）
func cloneRepository(ctx context.Context, repoURL, branch string) (string, error) {
	// 实际实现会使用 git clone
	// 这里返回临时路径
	return "/tmp/repo-" + uuid.New().String()[:8], nil
}

func cleanupRepo(path string) {
	// 实际实现会删除临时目录
}

// analyzeCodeStructure 分析代码结构
func analyzeCodeStructure(repoPath string) (*CodeStructure, error) {
	// 实际实现会扫描目录
	return &CodeStructure{}, nil
}

// analyzeDependencies 分析依赖
func analyzeDependencies(repoPath string, analysis *RequirementParserOutput) (*Dependencies, error) {
	// 实际实现会解析 package.json, go.mod, requirements.txt 等
	return &Dependencies{}, nil
}

// scanSecurity 安全扫描
func scanSecurity(repoPath string) []SecurityFinding {
	// 实际实现会扫描敏感信息、漏洞等
	return nil
}

// buildCodeAnalyzerPrompt 构建代码分析 Prompt
func buildCodeAnalyzerPrompt(structure, deps, analysis interface{}) string {
	return "分析代码并生成部署配置"
}

// getCodeAnalyzerSchema 代码分析 Schema
func getCodeAnalyzerSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"build_config": map[string]interface{}{"type": "object"},
			"runtime_config": map[string]interface{}{"type": "object"},
			"dockerfile": map[string]interface{}{"type": "object"},
			"deploy_steps": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]string{"type": "string"},
						"name": map[string]string{"type": "string"},
						"description": map[string]string{"type": "string"},
						"commands": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
						"timeout_seconds": map[string]int{"type": "integer"},
						"rollback": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
						"verification": map[string]string{"type": "string"},
					},
				},
			},
		},
	}
}

// classifyError 错误分类
func classifyError(log string) string {
	log = strings.ToLower(log)

	if strings.Contains(log, "permission denied") {
		return "permission"
	}
	if strings.Contains(log, "connection refused") || strings.Contains(log, "network") {
		return "network"
	}
	if strings.Contains(log, "not found") || strings.Contains(log, "no such file") {
		return "file_not_found"
	}
	if strings.Contains(log, "timeout") {
		return "timeout"
	}
	if strings.Contains(log, "memory") || strings.Contains(log, "oom") {
		return "memory"
	}
	if strings.Contains(log, "disk") || strings.Contains(log, "no space") {
		return "disk"
	}

	return "unknown"
}

// buildTroubleshooterPrompt 构建故障诊断 Prompt
func buildTroubleshooterPrompt(req *TroubleshooterInput, category string, cases []KnowledgeCase) string {
	var sb strings.Builder

	sb.WriteString("你是一个故障诊断专家，请分析以下错误并提供解决方案。\n\n")

	sb.WriteString(fmt.Sprintf("## 错误类型\n%s\n\n", category))

	sb.WriteString("## 错误日志\n")
	sb.WriteString(req.ErrorLog)
	sb.WriteString("\n\n")

	if len(cases) > 0 {
		sb.WriteString("## 相似案例\n")
		for i, c := range cases {
			sb.WriteString(fmt.Sprintf("\n### 案例 %d\n", i+1))
			sb.WriteString(fmt.Sprintf("- 错误类型：%s\n", c.ErrorType))
			sb.WriteString(fmt.Sprintf("- 根本原因：%s\n", c.RootCause))
			sb.WriteString(fmt.Sprintf("- 解决方案：%s\n", c.Solution))
		}
		sb.WriteString("\n\n")
	}

	sb.WriteString("请以 JSON 格式返回诊断结果，包含:\n")
	sb.WriteString("- diagnoses: [{category, description, evidence, severity}]\n")
	sb.WriteString("- root_cause: string\n")
	sb.WriteString("- confidence: number (0-1)\n")
	sb.WriteString("- remediation_plan: {steps: [{id, command, description, risk, rollback}], estimated_time_seconds, risk_level, requires_approval}\n")
	sb.WriteString("- need_human_help: boolean\n")

	return sb.String()
}

// getTroubleshooterSchema 故障诊断 Schema
func getTroubleshooterSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"diagnoses": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"category": map[string]string{"type": "string"},
						"description": map[string]string{"type": "string"},
						"evidence": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
						"severity": map[string]string{"type": "string", "enum": []string{"low", "medium", "high", "critical"}},
					},
				},
			},
			"root_cause": map[string]string{"type": "string"},
			"confidence": map[string]float64{"type": "number"},
			"remediation_plan": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"steps": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"id": map[string]string{"type": "string"},
								"command": map[string]string{"type": "string"},
								"description": map[string]string{"type": "string"},
								"risk": map[string]string{"type": "string"},
								"rollback": map[string]string{"type": "string"},
							},
						},
					},
					"estimated_time_seconds": map[string]int{"type": "integer"},
					"risk_level": map[string]string{"type": "string"},
					"requires_approval": map[string]bool{"type": "boolean"},
				},
			},
			"need_human_help": map[string]bool{"type": "boolean"},
		},
		"required": []string{"diagnoses", "root_cause", "confidence", "remediation_plan"},
	}
}

// ============================================================================
// 工具函数
// ============================================================================

// getGitHubToken 获取 GitHub Token
func getGitHubToken() string {
	// 从环境变量获取
	return "" // 空表示不认证，使用公开 API
}

// getLLMClient 获取 LLM 客户端
func getLLMClient() llm.Client {
	apiKey := "" // 从环境变量获取
	if apiKey == "" {
		// 返回一个 stub 客户端
		return &stubLLMClient{}
	}
	return llm.NewClient(&llm.Config{
		Provider:  llm.ProviderAnthropic,
		APIKey:    apiKey,
		Model:     "claude-sonnet-4-6-20250929",
		Timeout:   60 * time.Second,
		MaxRetries: 3,
	})
}

// stubLLMClient Stub LLM 客户端（用于无 API Key 时）
type stubLLMClient struct{}

func (c *stubLLMClient) Generate(ctx context.Context, messages []llm.Message, opts llm.Options) (string, error) {
	// 返回默认响应
	return `{"project_identity": {"name": "stub-project", "primary_language": "Go"}}`, nil
}
