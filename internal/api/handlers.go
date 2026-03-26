package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/servermind/aixm/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// register 用户注册
func (s *Server) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户是否已存在
	existing, _ := s.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// 创建用户（默认免费版）
	user := &models.User{
		ID:             uuid.New(),
		Email:          req.Email,
		Password:       string(hashedPassword),
		Name:           req.Name,
		Tier:           "free",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		MaxServers:     3,
		MaxDeployments: 10,
		MaxConcurrent:  1,
	}

	if err := s.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// 生成令牌
	accessToken, refreshToken, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"tier":  user.Tier,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// login 用户登录
func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户
	user, err := s.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// 生成令牌
	accessToken, refreshToken, err := s.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"tier":  user.Tier,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// refreshToken 刷新令牌
func (s *Server) refreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, err := s.jwtManager.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// getCurrentUser 获取当前用户
func (s *Server) getCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")

	user, err := s.userRepo.GetByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              user.ID,
		"email":           user.Email,
		"name":            user.Name,
		"avatar":          user.Avatar,
		"tier":            user.Tier,
		"max_servers":     user.MaxServers,
		"max_deployments": user.MaxDeployments,
		"max_concurrent":  user.MaxConcurrent,
		"created_at":      user.CreatedAt,
	})
}

// createServer 创建服务器
func (s *Server) createServer(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name     string `json:"name" binding:"required"`
		Host     string `json:"host" binding:"required,ip"`
		Port     int    `json:"port" binding:"required,min=1,max=65535"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户服务器配额
	servers, _ := s.serverRepo.ListByUserID(c.Request.Context(), userID.(uuid.UUID))
	user, _ := s.userRepo.GetByID(c.Request.Context(), userID.(uuid.UUID))
	if len(servers) >= user.MaxServers {
		c.JSON(http.StatusForbidden, gin.H{"error": "server limit reached"})
		return
	}

	// TODO: 将密码存入 Vault，此处简化处理
	server := &models.Server{
		ID:           uuid.New(),
		UserID:       userID.(uuid.UUID),
		Name:         req.Name,
		Host:         req.Host,
		Port:         req.Port,
		Username:     req.Username,
		CredentialID: req.Password, // TODO: 替换为 Vault 引用
		Status:       "offline",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.serverRepo.Create(c.Request.Context(), server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create server"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         server.ID,
		"name":       server.Name,
		"host":       server.Host,
		"port":       server.Port,
		"username":   server.Username,
		"status":     server.Status,
		"created_at": server.CreatedAt,
	})
}

// listServers 列出用户的所有服务器
func (s *Server) listServers(c *gin.Context) {
	userID, _ := c.Get("user_id")

	servers, err := s.serverRepo.ListByUserID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list servers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"servers": servers,
	})
}

// getServer 获取单个服务器
func (s *Server) getServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
		return
	}

	server, err := s.serverRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	c.JSON(http.StatusOK, server)
}

// updateServer 更新服务器
func (s *Server) updateServer(c *gin.Context) {
	// TODO: 实现更新逻辑
	c.JSON(http.StatusOK, gin.H{"message": "not implemented"})
}

// deleteServer 删除服务器
func (s *Server) deleteServer(c *gin.Context) {
	// TODO: 实现删除逻辑
	c.JSON(http.StatusOK, gin.H{"message": "not implemented"})
}

// connectServer 连接服务器
func (s *Server) connectServer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
		return
	}

	server, err := s.serverRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	// 尝试连接
	sshConfig := &ssh.SSHConfig{
		Host:     server.Host,
		Port:     server.Port,
		Username: server.Username,
		Password: server.CredentialID, // TODO: 从 Vault 获取
		Timeout:  s.config.SSH.ConnectTimeout,
	}

	client := ssh.NewSSHClient(sshConfig)
	ctx := c.Request.Context()

	if err := client.Connect(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "failed to connect to server",
			"details": err.Error(),
		})
		return
	}
	defer client.Close()

	// 获取服务器信息
	result, err := client.Execute(ctx, "uname -a", ssh.ExecuteOptions{})
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "failed to get server info",
		})
		return
	}

	// 更新服务器状态
	s.serverRepo.UpdateStatus(c.Request.Context(), id, "online", time.Now())

	c.JSON(http.StatusOK, gin.H{
		"status": "connected",
		"info":   result.Output,
	})
}

// getServerStatus 获取服务器状态
func (s *Server) getServerStatus(c *gin.Context) {
	// TODO: 实现
	c.JSON(http.StatusOK, gin.H{"message": "not implemented"})
}
