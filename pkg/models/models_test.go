package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	user := &User{
		ID:             uuid.New(),
		Email:          "test@example.com",
		Password:       "hashed-password",
		Name:           "Test User",
		Tier:           "free",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		MaxServers:     3,
		MaxDeployments: 10,
		MaxConcurrent:  1,
	}

	if user.ID == uuid.Nil {
		t.Error("Expected user ID to be set")
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
	}
	if user.Tier != "free" {
		t.Errorf("Expected tier 'free', got '%s'", user.Tier)
	}
	if user.MaxServers != 3 {
		t.Errorf("Expected max servers 3, got %d", user.MaxServers)
	}
}

func TestUserTierValues(t *testing.T) {
	tiers := []string{"free", "pro", "team", "enterprise"}

	for _, tier := range tiers {
		user := &User{
			ID:    uuid.New(),
			Email: "test@example.com",
			Name:  "Test",
			Tier:  tier,
		}

		if user.Tier != tier {
			t.Errorf("Expected tier '%s', got '%s'", tier, user.Tier)
		}
	}
}

func TestNewServer(t *testing.T) {
	userID := uuid.New()
	server := &Server{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         "Production Server",
		Host:         "192.168.1.100",
		Port:         22,
		Username:     "root",
		CredentialID: "vault-ref-123",
		Status:       "offline",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if server.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, server.UserID)
	}
	if server.Host != "192.168.1.100" {
		t.Errorf("Expected host '192.168.1.100', got '%s'", server.Host)
	}
	if server.Port != 22 {
		t.Errorf("Expected port 22, got %d", server.Port)
	}
	if server.Status != "offline" {
		t.Errorf("Expected status 'offline', got '%s'", server.Status)
	}
}

func TestServerStatusValues(t *testing.T) {
	statuses := []ServerStatus{ServerStatusOnline, ServerStatusOffline, ServerStatusError}
	expected := []string{"online", "offline", "error"}

	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("Expected status '%s', got '%s'", expected[i], status)
		}
	}
}

func TestNewDeployment(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	projectID := uuid.New()

	deployment := &Deployment{
		ID:          uuid.New(),
		UserID:      userID,
		ServerID:    serverID,
		ProjectID:   projectID,
		ProjectName: "My Project",
		RepoURL:     "https://github.com/user/repo",
		Branch:      "main",
		Status:      "pending",
		State:       "initialized",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if deployment.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, deployment.UserID)
	}
	if deployment.RepoURL != "https://github.com/user/repo" {
		t.Errorf("Expected repo URL 'https://github.com/user/repo', got '%s'", deployment.RepoURL)
	}
	if deployment.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", deployment.Status)
	}
}

func TestDeployResult(t *testing.T) {
	result := &DeployResult{
		Success: true,
		Endpoints: []ServiceEndpoint{
			{Port: 8080, Protocol: "http", Purpose: "API", Path: "/api"},
			{Port: 443, Protocol: "https", Purpose: "Web"},
		},
		Steps: []DeployStep{
			{ID: "1", Name: "Build", Status: "success"},
			{ID: "2", Name: "Deploy", Status: "success"},
		},
		Warnings: []string{"Low memory warning"},
	}

	if !result.Success {
		t.Error("Expected success to be true")
	}
	if len(result.Endpoints) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(result.Endpoints))
	}
	if len(result.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(result.Steps))
	}
}

func TestDeployErrorSeverity(t *testing.T) {
	severities := []string{"critical", "error", "warning"}

	for _, severity := range severities {
		err := &DeployError{
			Code:     "TEST_001",
			Message:  "Test error",
			Severity: severity,
		}

		if err.Severity != severity {
			t.Errorf("Expected severity '%s', got '%s'", severity, err.Severity)
		}
	}
}

func TestNewProject(t *testing.T) {
	userID := uuid.New()
	project := &Project{
		ID:               uuid.New(),
		UserID:           userID,
		Name:             "Test Project",
		RepoURL:          "https://github.com/user/test-project",
		RepoOwner:        "user",
		RepoName:         "test-project",
		DefaultBranch:    "main",
		Language:         "Go",
		FrameWork:        "Gin",
		DeployType:       "docker",
		HasDockerfile:    true,
		HasDockerCompose: true,
		MinCPU:           1.0,
		MinMemory:        1024,
		MinDisk:          10,
		ExposedPorts:     []int{8080, 443},
		EnvVars: []EnvVar{
			{Name: "PORT", Required: true, Default: "8080"},
			{Name: "DB_PASSWORD", Required: true, Sensitive: true},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if project.RepoOwner != "user" {
		t.Errorf("Expected repo owner 'user', got '%s'", project.RepoOwner)
	}
	if !project.HasDockerfile {
		t.Error("Expected HasDockerfile to be true")
	}
	if len(project.ExposedPorts) != 2 {
		t.Errorf("Expected 2 exposed ports, got %d", len(project.ExposedPorts))
	}
}

func TestEnvVarSensitive(t *testing.T) {
	envVar := EnvVar{
		Name:        "API_KEY",
		Description: "API key for external service",
		Required:    true,
		Sensitive:   true,
	}

	if !envVar.Sensitive {
		t.Error("Expected envVar to be marked as sensitive")
	}
	if !envVar.Required {
		t.Error("Expected envVar to be required")
	}
}

func TestKnowledgeCase(t *testing.T) {
	kc := &KnowledgeCase{
		ID:           uuid.New(),
		CaseType:     "success",
		OS:           "Ubuntu",
		OSVersion:    "22.04",
		TechStack:    "Go,Gin,PostgreSQL",
		Runtime:      "Go 1.21",
		Solution:     "Updated configuration file",
		Commands:     []string{"systemctl restart app", "journalctl -u app"},
		SuccessCount: 10,
		FailureCount: 0,
		QualityScore: 0.95,
		IsActive:     true,
	}

	if kc.CaseType != "success" {
		t.Errorf("Expected case type 'success', got '%s'", kc.CaseType)
	}
	if kc.QualityScore != 0.95 {
		t.Errorf("Expected quality score 0.95, got %f", kc.QualityScore)
	}
	if len(kc.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(kc.Commands))
	}
}

func TestAuditLog(t *testing.T) {
	userID := uuid.New()
	resourceID := uuid.New()

	log := &AuditLog{
		ID:         uuid.New(),
		UserID:     userID,
		Action:     "CREATE_SERVER",
		Resource:   "server",
		ResourceID: resourceID,
		Details:    "Created new production server",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		Success:    true,
	}

	if log.Action != "CREATE_SERVER" {
		t.Errorf("Expected action 'CREATE_SERVER', got '%s'", log.Action)
	}
	if !log.Success {
		t.Error("Expected success to be true")
	}
	if log.IPAddress != "192.168.1.1" {
		t.Errorf("Expected IP address '192.168.1.1', got '%s'", log.IPAddress)
	}
}
