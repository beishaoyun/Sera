package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/servermind/aixm/internal/auth"
	"github.com/servermind/aixm/internal/database"
	"github.com/servermind/aixm/internal/ssh"
	"github.com/servermind/aixm/internal/config"
	"github.com/servermind/aixm/internal/llm"
	"github.com/google/uuid"
)

// Server HTTP API 服务器
type Server struct {
	*gin.Engine
	config        *config.Config
	userRepo      *database.UserRepository
	serverRepo    *database.ServerRepository
	deployRepo    *database.DeploymentRepository
	sshPool       *ssh.SSHPool
	jwtManager    *auth.JWTManager
	llmClient     llm.LLMClient
}

// NewServer 创建 API 服务器
func NewServer(cfg *config.Config, db *database.Database) (*Server, error) {
	// 设置 Gin 模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.Default()

	// 初始化仓库
	userRepo := database.NewUserRepository(db)
	serverRepo := database.NewServerRepository(db)
	deployRepo := database.NewDeploymentRepository(db)

	// 初始化 JWT 管理器
	jwtManager := auth.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.TokenExpiry,
		cfg.Auth.RefreshExpiry,
	)

	// 初始化 SSH 连接池
	sshConfig := &ssh.SSHConfig{
		Timeout:   cfg.SSH.ConnectTimeout,
		KeepAlive: cfg.SSH.KeepAlive,
	}
	sshPool := ssh.NewSSHPool(cfg.SSH.MaxConnections, sshConfig)

	// 初始化 LLM 客户端
	llmClient := initLLMClient(cfg)

	s := &Server{
		Engine:     engine,
		config:     cfg,
		userRepo:   userRepo,
		serverRepo: serverRepo,
		deployRepo: deployRepo,
		sshPool:    sshPool,
		jwtManager: jwtManager,
		llmClient:  llmClient,
	}

	// 注册路由
	s.registerRoutes()

	return s, nil
}

// registerRoutes 注册路由
func (s *Server) registerRoutes() {
	// 健康检查
	s.GET("/health", s.healthCheck)
	s.GET("/ready", s.readyCheck)

	// API v1
	v1 := s.Group("/api/v1")
	{
		// 公开路由
		auth := v1.Group("/auth")
		{
			auth.POST("/register", s.register)
			auth.POST("/login", s.login)
			auth.POST("/refresh", s.refreshToken)
		}

		// 需要认证的路由
		protected := v1.Group("")
		protected.Use(s.authMiddleware())
		{
			// 用户
			protected.GET("/user", s.getCurrentUser)
			protected.PUT("/user", s.updateUser)

			// 服务器
			servers := protected.Group("/servers")
			{
				servers.POST("", s.createServer)
				servers.GET("", s.listServers)
				servers.GET("/:id", s.getServer)
				servers.PUT("/:id", s.updateServer)
				servers.DELETE("/:id", s.deleteServer)
				servers.POST("/:id/connect", s.connectServer)
				servers.GET("/:id/status", s.getServerStatus)
			}

			// 部署
			deployments := protected.Group("/deployments")
			{
				deployments.POST("", s.createDeployment)
				deployments.GET("", s.listDeployments)
				deployments.GET("/:id", s.getDeployment)
				deployments.POST("/:id/cancel", s.cancelDeployment)
				deployments.GET("/:id/logs", s.getDeploymentLogs)
			}

			// 项目
			projects := protected.Group("/projects")
			{
				projects.POST("", s.createProject)
				projects.GET("", s.listProjects)
				projects.GET("/:id", s.getProject)
				projects.DELETE("/:id", s.deleteProject)
			}
		}
	}

	// Swagger (仅 debug 模式)
	if s.config.Server.EnableSwagger {
		// TODO: 添加 Swagger 支持
		// s.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 内容解析 API（支持 GitHub 和教程）
	v1.POST("/parse-content", s.parseContent)
}

// initLLMClient 初始化 LLM 客户端
func initLLMClient(cfg *config.Config) llm.LLMClient {
	// 从环境变量获取配置
	apiKey := getEnv("LLM_API_KEY", "")
	provider := getEnv("LLM_PROVIDER", "openai")
	model := getEnv("LLM_MODEL", "")
	baseURL := getEnv("LLM_BASE_URL", "")

	if apiKey == "" {
		// 返回一个 mock 客户端
		return &MockLLMClient{}
	}

	switch provider {
	case "claude":
		return llm.NewClaudeClient(apiKey, model)
	case "ollama":
		return llm.NewOpenAIClient(apiKey, baseURL, model)
	default:
		return llm.NewOpenAIClient(apiKey, baseURL, model)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// MockLLMClient Mock LLM 客户端（当没有配置 API 时使用）
type MockLLMClient struct{}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string, opts llm.GenerateOptions) (string, error) {
	return "{}", nil
}
func (m *MockLLMClient) GenerateJSON(ctx context.Context, prompt string, schema map[string]interface{}, opts llm.GenerateOptions) (interface{}, error) {
	return map[string]interface{}{}, nil
}
func (m *MockLLMClient) Chat(ctx context.Context, messages []llm.Message, opts llm.GenerateOptions) (string, error) {
	return "{}", nil
}
func (m *MockLLMClient) AnalyzeProject(ctx context.Context, repoURL, readmeContent, fileTree string) (*llm.ProjectAnalysis, error) {
	return &llm.ProjectAnalysis{Name: "project"}, nil
}
func (m *MockLLMClient) ParseTutorial(ctx context.Context, tutorialContent string) (*llm.TutorialSteps, error) {
	return &llm.TutorialSteps{Title: "tutorial"}, nil
}
func (m *MockLLMClient) GenerateTroubleshooting(ctx context.Context, errorLog, context string) (*llm.TroubleshootingResult, error) {
	return &llm.TroubleshootingResult{}, nil
}

// authMiddleware JWT 认证中间件
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// 提取 Bearer Token
		var tokenString string
		_, err := fmt.Sscanf(authHeader, "Bearer %s", &tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		// 验证令牌
		claims, err := s.jwtManager.VerifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("tier", claims.Tier)

		c.Next()
	}
}

// healthCheck 健康检查
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// readyCheck 就绪检查
func (s *Server) readyCheck(c *gin.Context) {
	// TODO: 检查数据库、Redis 等依赖服务
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// Start 启动服务器
func (s *Server) Start() error {
	addr := ":" + s.config.Server.Port
	return s.Engine.Run(addr)
}
