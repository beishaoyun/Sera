package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/servermind/aixm/internal/llm"
	"github.com/servermind/aixm/internal/scraper"
	"github.com/servermind/aixm/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// CreateDeploymentRequest 创建部署请求
type CreateDeploymentRequest struct {
	ServerID    string `json:"server_id" binding:"required"`
	ProjectID   string `json:"project_id,omitempty"`
	RepoURL     string `json:"repo_url,omitempty"`     // GitHub 项目 URL
	TutorialURL string `json:"tutorial_url,omitempty"` // 教程 URL
	Branch      string `json:"branch,omitempty"`
	Content     string `json:"content,omitempty"` // 手动输入的内容
}

// createDeployment 创建部署
func (s *Server) createDeployment(c *gin.Context) {
	var req CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证：必须有 repo_url 或 tutorial_url 之一
	if req.RepoURL == "" && req.TutorialURL == "" && req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "repo_url, tutorial_url, or content is required",
		})
		return
	}

	userID, _ := c.Get("user_id")
	userIDStr := userID.(uuid.UUID).String()

	// 解析输入来源
	var (
		projectName string
		content     string
		sourceType  string // github, tutorial, manual
	)

	ctx := c.Request.Context()

	if req.RepoURL != "" {
		// GitHub 项目部署
		sourceType = "github"
		projectName = extractRepoName(req.RepoURL)

		// 抓取 README
		scraper := scraper.NewTutorialScraper()
		readmeContent, err := scraper.ScrapeGitHubReadme(ctx, req.RepoURL)
		if err != nil {
			// 如果 README 抓取失败，继续但记录警告
			content = fmt.Sprintf("Failed to fetch README: %v", err)
		} else {
			content = readmeContent
		}

	} else if req.TutorialURL != "" {
		// 教程部署
		sourceType = "tutorial"
		projectName = "tutorial-deployment"

		scraper := scraper.NewTutorialScraper()
		scrapedContent, err := scraper.Scrape(ctx, req.TutorialURL)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Failed to fetch tutorial: %v", err),
			})
			return
		}

		projectName = scrapedContent.Title
		content = scrapedContent.Content

	} else {
		// 手动输入内容
		sourceType = "manual"
		projectName = "manual-deployment"
		content = req.Content
	}

	// 使用 LLM 分析内容并生成部署配置
	llmClient := s.getLLMClient()

	var deployConfig *llm.ProjectAnalysis
	var tutorialSteps *llm.TutorialSteps

	if sourceType == "github" {
		// 分析 GitHub 项目
		var err error
		deployConfig, err = llmClient.AnalyzeProject(ctx, req.RepoURL, content, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to analyze project: %v", err),
			})
			return
		}
	} else {
		// 解析教程
		var err error
		tutorialSteps, err = llmClient.ParseTutorial(ctx, content)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to parse tutorial: %v", err),
			})
			return
		}
	}

	// 创建项目记录
	project := &models.Project{
		ID:            uuid.New(),
		UserID:        userID.(uuid.UUID),
		Name:          projectName,
		RepoURL:       req.RepoURL,
		DefaultBranch: req.Branch,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 创建工作流输入
	workflowInput := map[string]interface{}{
		"user_id":       userIDStr,
		"server_id":     req.ServerID,
		"project_id":    project.ID.String(),
		"source_type":   sourceType,
		"content":       content,
		"deploy_config": deployConfig,
		"tutorial":      tutorialSteps,
	}

	// 创建部署记录
	deployment := &models.Deployment{
		ID:          uuid.New(),
		UserID:      userID.(uuid.UUID),
		ServerID:    uuid.MustParse(req.ServerID),
		ProjectID:   project.ID,
		ProjectName: projectName,
		RepoURL:     req.RepoURL,
		Branch:      req.Branch,
		Status:      "pending",
		State:       "PENDING",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 启动部署工作流
	workflowID, err := s.startDeploymentWorkflow(ctx, workflowInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to start workflow: %v", err),
		})
		return
	}

	deployment.WorkflowID = workflowID

	// 保存到数据库
	if err := s.deployRepo.Create(ctx, deployment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create deployment record",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           deployment.ID,
		"project_name": projectName,
		"source_type":  sourceType,
		"status":       deployment.Status,
		"workflow_id":  workflowID,
		"created_at":   deployment.CreatedAt,
	})
}

// parseContent 解析内容（GitHub 或教程）
type ParseContentRequest struct {
	RepoURL     string `json:"repo_url,omitempty"`
	TutorialURL string `json:"tutorial_url,omitempty"`
}

type ParseContentResponse struct {
	ProjectName    string                 `json:"project_name"`
	SourceType     string                 `json:"source_type"`
	ProjectInfo    *llm.ProjectAnalysis   `json:"project_info,omitempty"`
	TutorialInfo   *llm.TutorialSteps     `json:"tutorial_info,omitempty"`
	RawContent     string                 `json:"raw_content,omitempty"`
}

// parseContent 解析 GitHub 项目或教程（不执行部署）
func (s *Server) parseContent(c *gin.Context) {
	var req ParseContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	llmClient := s.getLLMClient()
	scraper := scraper.NewTutorialScraper()

	response := &ParseContentResponse{}

	if req.RepoURL != "" {
		response.SourceType = "github"

		// 抓取 README
		readmeContent, err := scraper.ScrapeGitHubReadme(ctx, req.RepoURL)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Failed to fetch README: %v", err),
			})
			return
		}

		response.RawContent = readmeContent

		// 分析项目
		projectInfo, err := llmClient.AnalyzeProject(ctx, req.RepoURL, readmeContent, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to analyze project: %v", err),
			})
			return
		}

		response.ProjectName = projectInfo.Name
		response.ProjectInfo = projectInfo

	} else if req.TutorialURL != "" {
		response.SourceType = "tutorial"

		// 抓取教程
		scrapedContent, err := scraper.Scrape(ctx, req.TutorialURL)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Failed to fetch tutorial: %v", err),
			})
			return
		}

		response.RawContent = scrapedContent.Content
		response.ProjectName = scrapedContent.Title

		// 解析教程
		tutorialSteps, err := llmClient.ParseTutorial(ctx, scrapedContent.Content)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to parse tutorial: %v", err),
			})
			return
		}

		response.TutorialInfo = tutorialSteps
	}

	c.JSON(http.StatusOK, response)
}

// getLLMClient 获取 LLM 客户端
func (s *Server) getLLMClient() llm.LLMClient {
	// 从配置获取 LLM 设置
	// 这里简化处理，默认使用 OpenAI
	apiKey := "your-api-key" // 应该从配置或 Vault 获取
	baseURL := ""
	model := ""

	return llm.NewOpenAIClient(apiKey, baseURL, model)
}

// startDeploymentWorkflow 启动部署工作流
func (s *Server) startDeploymentWorkflow(ctx context.Context, input map[string]interface{}) (string, error) {
	// 简化实现，实际应该使用 Temporal
	workflowID := fmt.Sprintf("deployment-%s", uuid.New().String()[:8])

	// 在后台启动 goroutine 执行部署
	go s.executeDeployment(ctx, workflowID, input)

	return workflowID, nil
}

// executeDeployment 执行部署（后台运行）
func (s *Server) executeDeployment(ctx context.Context, workflowID string, input map[string]interface{}) {
	// 1. 获取服务器连接信息
	serverID := input["server_id"].(string)
	// ... 获取服务器详情

	// 2. 根据来源类型执行不同流程
	sourceType := input["source_type"].(string)

	if sourceType == "github" {
		// GitHub 项目部署流程
		s.deployFromGitHub(ctx, input)
	} else {
		// 教程部署流程
		s.deployFromTutorial(ctx, input)
	}

	// 3. 更新部署状态
	// 4. 学习并存储到知识库
}

// deployFromGitHub 从 GitHub 项目部署
func (s *Server) deployFromGitHub(ctx context.Context, input map[string]interface{}) {
	// 使用 agent 系统执行部署
	// 1. RequirementParser
	// 2. CodeAnalyzer
	// 3. DeploymentExecutor
	// 4. Troubleshooter (如有错误)
}

// deployFromTutorial 从教程部署
func (s *Server) deployFromTutorial(ctx context.Context, input map[string]interface{}) {
	tutorial := input["tutorial_info"].(*llm.TutorialSteps)

	// 按教程步骤执行
	for _, step := range tutorial.Steps {
		if step.Manual {
			// 需要手动确认的步骤
			// 暂停等待用户确认
		} else {
			// 执行命令
			// s.executeCommand(step.Command)
		}
	}
}

// extractRepoName 从 URL 提取仓库名
func extractRepoName(url string) string {
	// https://github.com/owner/repo -> owner/repo
	// 简化实现
	return url
}
