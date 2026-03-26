package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/servermind/aixm/internal/agent"
	"github.com/servermind/aixm/internal/asciinema"
	"github.com/servermind/aixm/internal/github"
	"github.com/servermind/aixm/internal/knowledge"
	"github.com/servermind/aixm/internal/sandbox"
	"github.com/servermind/aixm/internal/temporal"
	"github.com/servermind/aixm/internal/vault"
	"github.com/servermind/aixm/internal/websocket"
	"github.com/servermind/aixm/pkg/models"
)

// ============================================================================
// 应用上下文
// ============================================================================

// AppContext 应用上下文
type AppContext struct {
	TemporalClient   *temporal.Client
	KnowledgeManager *knowledge.KnowledgeCaseManager
	WSHandler        *websocket.WSHandler
	AsciinemaRecorder *asciinema.Recorder
	VaultClient      *vault.Client
	GitHubClient     *github.Client
	Sandbox          *sandbox.CommandSandbox
}

// ============================================================================
// 部署处理器
// ============================================================================

// DeploymentHandler 部署处理器
type DeploymentHandler struct {
	app *AppContext
}

// NewDeploymentHandler 创建部署处理器
func NewDeploymentHandler(app *AppContext) *DeploymentHandler {
	return &DeploymentHandler{
		app: app,
	}
}

// CreateDeployment 创建部署
// POST /api/deployments
func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
	var req struct {
		RepoURL     string `json:"repo_url" binding:"required"`
		ServerID    string `json:"server_id" binding:"required"`
		Branch      string `json:"branch"`
		UserRequest string `json:"user_request"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 GitHub URL
	repoInfo, err := github.ParseURL(req.RepoURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GitHub URL"})
		return
	}

	// 创建部署记录
	deployment := &models.Deployment{
		ID:         uuid.New(),
		UserID:     uuid.Nil, // TODO: 从 JWT 获取用户 ID
		ServerID:   uuid.MustParse(req.ServerID),
		ProjectID:  uuid.Nil,
		ProjectName: fmt.Sprintf("%s/%s", repoInfo.Owner, repoInfo.Repo),
		RepoURL:    req.RepoURL,
		Branch:     req.Branch,
		Status:     "pending",
		State:      "PENDING",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// TODO: 保存到数据库
	// if err := db.Save(deployment); err != nil { ... }

	// 启动工作流
	workflowInput := temporal.DeploymentWorkflowInput{
		DeploymentID: deployment.ID.String(),
		UserID:       deployment.UserID.String(),
		RepoURL:      req.RepoURL,
		Branch:       req.Branch,
		ServerID:     req.ServerID,
	}

	workflowID, err := h.app.TemporalClient.StartDeploymentWorkflow(c, workflowInput)
	if err != nil {
		logrus.Errorf("Failed to start workflow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start deployment"})
		return
	}

	// 更新部署记录
	deployment.WorkflowID = workflowID

	c.JSON(http.StatusCreated, gin.H{
		"deployment_id": deployment.ID,
		"workflow_id":   workflowID,
		"status":        "pending",
	})
}

// GetDeployment 获取部署状态
// GET /api/deployments/:id
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	deploymentID := c.Param("id")

	// TODO: 从数据库获取部署信息

	c.JSON(http.StatusOK, gin.H{
		"id":     deploymentID,
		"status": "running",
		"state":  "CODE_FETCHING",
	})
}

// GetDeploymentProgress 获取部署进度
// GET /api/deployments/:id/progress
func (h *DeploymentHandler) GetDeploymentProgress(c *gin.Context) {
	deploymentID := c.Param("id")

	// TODO: 查询 Temporal 工作流状态

	c.JSON(http.StatusOK, gin.H{
		"deployment_id": deploymentID,
		"progress":      45,
		"current_step":  "Installing dependencies",
		"state":         "DEPENDENCY_INSTALLING",
		"logs":          "npm install completed",
	})
}

// CancelDeployment 取消部署
// POST /api/deployments/:id/cancel
func (h *DeploymentHandler) CancelDeployment(c *gin.Context) {
	deploymentID := c.Param("id")

	// TODO: 取消 Temporal 工作流

	c.JSON(http.StatusOK, gin.H{
		"message": "Deployment cancelled",
	})
}

// RetryDeployment 重试部署
// POST /api/deployments/:id/retry
func (h *DeploymentHandler) RetryDeployment(c *gin.Context) {
	deploymentID := c.Param("id")

	// TODO: 重试部署

	c.JSON(http.StatusOK, gin.H{
		"message": "Deployment retrying",
	})
}

// ============================================================================
// WebSocket 处理器
// ============================================================================

// WSHub 全局 WebSocket Hub
var WSHub *websocket.WSHandler

// InitWebSocket 初始化 WebSocket
func InitWebSocket() *websocket.WSHandler {
	WSHub = websocket.NewWSHandler()
	return WSHub
}

// HandleWebSocket 处理 WebSocket 连接
// WS /ws/deployments/:id
func HandleWebSocket(c *gin.Context) {
	if WSHub == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket not initialized"})
		return
	}
	WSHub.HandleWebSocket(c)
}

// ============================================================================
// 服务器处理器
// ============================================================================

// ServerHandler 服务器处理器
type ServerHandler struct {
	app *AppContext
}

// NewServerHandler 创建服务器处理器
func NewServerHandler(app *AppContext) *ServerHandler {
	return &ServerHandler{app: app}
}

// CreateServer 创建服务器
// POST /api/servers
func (h *ServerHandler) CreateServer(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Host     string `json:"host" binding:"required"`
		Port     int    `json:"port" binding:"required"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password"`
		SSHKey   string `json:"ssh_key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建 SSH 证书（通过 Vault）
	if h.app.VaultClient != nil {
		// 使用 Vault 生成动态 SSH 证书
		certReq := &vault.SignSSHKeyRequest{
			KeyID:      fmt.Sprintf("deploy-%s", uuid.New().String()[:8]),
			PublicKey:  req.SSHKey,
			Principals: []string{req.Username, "root"},
			TTL:        24 * time.Hour,
			Role:       "deploy-user",
		}

		cert, err := h.app.VaultClient.SignSSHKey(c, certReq)
		if err != nil {
			logrus.Errorf("Failed to sign SSH key: %v", err)
		} else {
			// 存储证书
			h.app.VaultClient.StoreSSHCertificate(c, uuid.New().String(), cert)
		}
	}

	server := &models.Server{
		ID:        uuid.New(),
		UserID:    uuid.Nil, // TODO: 从 JWT 获取
		Name:      req.Name,
		Host:      req.Host,
		Port:      req.Port,
		Username:  req.Username,
		Status:    "online",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// TODO: 保存到数据库

	c.JSON(http.StatusCreated, gin.H{
		"id":     server.ID,
		"status": "created",
	})
}

// ListServers 列出服务器
// GET /api/servers
func (h *ServerHandler) ListServers(c *gin.Context) {
	// TODO: 从数据库获取

	c.JSON(http.StatusOK, []models.Server{})
}

// GetServer 获取服务器详情
// GET /api/servers/:id
func (h *ServerHandler) GetServer(c *gin.Context) {
	serverID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{
		"id":     serverID,
		"status": "online",
	})
}

// DeleteServer 删除服务器
// DELETE /api/servers/:id
func (h *ServerHandler) DeleteServer(c *gin.Context) {
	serverID := c.Param("id")

	// TODO: 删除服务器

	c.JSON(http.StatusOK, gin.H{
		"message": "Server deleted",
	})
}

// TestServerConnection 测试服务器连接
// POST /api/servers/:id/test
func (h *ServerHandler) TestServerConnection(c *gin.Context) {
	serverID := c.Param("id")

	// TODO: 实际测试 SSH 连接

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Connection successful",
	})
}

// ============================================================================
// 知识库处理器
// ============================================================================

// KnowledgeHandler 知识库处理器
type KnowledgeHandler struct {
	app *AppContext
}

// NewKnowledgeHandler 创建知识库处理器
func NewKnowledgeHandler(app *AppContext) *KnowledgeHandler {
	return &KnowledgeHandler{app: app}
}

// SearchKnowledge 搜索知识库
// GET /api/knowledge/search?q=error
func (h *KnowledgeHandler) SearchKnowledge(c *gin.Context) {
	query := c.Query("q")
	limit := 10

	if h.app.KnowledgeManager == nil {
		c.JSON(http.StatusOK, gin.H{"results": []interface{}{}})
		return
	}

	results, err := h.app.KnowledgeManager.FindSimilarCases(c, query, "", limit)
	if err != nil {
		logrus.Errorf("Failed to search knowledge: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

// ListKnowledgeCases 列出知识案例
// GET /api/knowledge/cases
func (h *KnowledgeHandler) ListKnowledgeCases(c *gin.Context) {
	caseType := c.Query("type") // success, failure

	// TODO: 从数据库获取

	c.JSON(http.StatusOK, gin.H{
		"cases": []interface{}{},
	})
}

// ============================================================================
// 健康检查
// ============================================================================

// HealthCheck 健康检查
// GET /health
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ReadyCheck 就绪检查
// GET /ready
func ReadyCheck(c *gin.Context) {
	// 检查依赖服务
	health := gin.H{
		"status": "ready",
		"checks": gin.H{},
	}

	// TODO: 检查数据库、Redis、Milvus、Temporal 等

	c.JSON(http.StatusOK, health)
}
