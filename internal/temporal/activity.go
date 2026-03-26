package temporal

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/servermind/aixm/internal/agent"
	"github.com/servermind/aixm/internal/sandbox"
	"github.com/servermind/aixm/internal/ssh"
	"github.com/servermind/aixm/pkg/errors"
)

// ============================================================================
// Activity 实现 - 调用实际 Agent
// ============================================================================

// RequirementParserActivity 需求解析活动
func RequirementParserActivity(ctx context.Context, input RequirementParserInput) (*RequirementParserResult, error) {
	log := logrus.WithContext(ctx)
	log.Info("Executing requirement parser activity")

	// 创建 Agent（实际实现需要 LLM 客户端）
	// llmClient := NewLLMClient(...)
	// parser := agent.NewRequirementParser(llmClient)

	// 桩实现 - 实际会调用 LLM
	time.Sleep(2 * time.Second)

	return &RequirementParserResult{
		Output: RequirementParserOutput{
			ProjectIdentity: ProjectIdentity{
				Name:            "example-project",
				PrimaryLanguage: "Go",
			},
			DeploymentProfile: DeploymentProfile{
				Type:            "go_application",
				BuildRequired:   true,
				Runtime:         "go1.21",
				PackageManager:  "go mod",
				EstimatedComplexity: "medium",
			},
		},
	}, nil
}

// CodeAnalyzerActivity 代码分析活动
func CodeAnalyzerActivity(ctx context.Context, input CodeAnalyzerInput) (*CodeAnalyzerResult, error) {
	log := logrus.WithContext(ctx)
	log.Info("Executing code analyzer activity")

	// 桩实现 - 实际会调用 LLM 分析代码
	time.Sleep(2 * time.Second)

	return &CodeAnalyzerResult{
		DeploySteps: []DeployStep{
			{
				ID:          "step1",
				Name:        "Clone repository",
				Description: "Clone the repository from GitHub",
				Commands:    []string{"git clone https://github.com/example/repo.git /app"},
				Timeout:     300,
				Rollback:    []string{"rm -rf /app"},
			},
			{
				ID:          "step2",
				Name:        "Install dependencies",
				Description: "Install Go modules",
				Commands:    []string{"cd /app && go mod download"},
				Timeout:     120,
			},
			{
				ID:          "step3",
				Name:        "Build",
				Description: "Build the Go application",
				Commands:    []string{"cd /app && go build -o app ./cmd/server"},
				Timeout:     180,
				Verification: "test -f /app/app",
			},
			{
				ID:          "step4",
				Name:        "Deploy",
				Description: "Start the application",
				Commands:    []string{"cd /app && ./app &"},
				Timeout:     60,
				Verification: "curl -f http://localhost:8080/health || true",
			},
		},
	}, nil
}

// DeploymentExecutorActivity 部署执行活动（调用实际 Agent）
func DeploymentExecutorActivity(ctx context.Context, input DeploymentExecutorInput) (*DeploymentExecutorResult, error) {
	log := logrus.WithContext(ctx)
	log.WithFields(logrus.Fields{
		"server_id": input.ServerID,
		"steps":     len(input.DeploySteps),
	}).Info("Executing deployment executor activity")

	// 创建命令沙箱
	sandboxConfig := sandbox.DefaultSandboxConfig()
	sb := sandbox.NewCommandSandbox(sandboxConfig)

	// 创建 SSH 工厂（需要实现 ServerConfigFetcher）
	// sshFactory := ssh.NewDatabaseSSHClientFactory(fetcher, sb)

	// 临时使用简单工厂
	sshFactory := ssh.NewSSHClientFactory(nil, sb)
	defer sshFactory.Close()

	// 创建 Agent
	executor := agent.NewDeploymentExecutor(sshFactory)

	// 执行
	result, err := executor.Execute(ctx, &input)
	if err != nil {
		log.Errorf("Deployment executor failed: %v", err)

		appErr, ok := err.(*errors.AppError)
		if ok && appErr.Code == errors.ErrCommandBlocked {
			return &DeploymentExecutorResult{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Command blocked by security sandbox: %s", appErr.Message),
				Errors: []DeployError{
					{
						Code:     "COMMAND_BLOCKED",
						Message:  appErr.Message,
						Severity: "critical",
					},
				},
			}, nil
		}

		return &DeploymentExecutorResult{
			Success:      false,
			ErrorMessage: err.Error(),
			Errors: []DeployError{
				{
					Code:     "EXECUTION_ERROR",
					Message:  err.Error(),
					Severity: "error",
				},
			},
		}, nil
	}

	// 类型断言
	execResult, ok := result.(*DeploymentExecutorOutput)
	if !ok {
		return nil, fmt.Errorf("unexpected result type from DeploymentExecutor")
	}

	// 转换为活动输出类型
	activityResult := &DeploymentExecutorResult{
		Success: execResult.Success,
		Steps:   make([]StepExecutionResult, len(execResult.Steps)),
	}

	for i, step := range execResult.Steps {
		activityResult.Steps[i] = StepExecutionResult{
			StepID: step.StepID,
			Status: step.Status,
		}
	}

	if len(execResult.Endpoints) > 0 {
		activityResult.Endpoints = make([]ServiceEndpoint, len(execResult.Endpoints))
		for i, ep := range execResult.Endpoints {
			activityResult.Endpoints[i] = ServiceEndpoint{
				Port:     ep.Port,
				Protocol: ep.Protocol,
				Path:     ep.Path,
			}
		}
	}

	if len(execResult.Errors) > 0 {
		activityResult.Errors = make([]DeployError, len(execResult.Errors))
		for i, e := range execResult.Errors {
			activityResult.Errors[i] = DeployError{
				Code:     e.Code,
				Message:  e.Message,
				Severity: e.Severity,
			}
		}
	}

	return activityResult, nil
}

// TroubleshooterActivity 故障诊断活动
func TroubleshooterActivity(ctx context.Context, input TroubleshooterInput) (*TroubleshooterResult, error) {
	log := logrus.WithContext(ctx)
	log.WithField("error_log", input.ErrorLog).Info("Executing troubleshooter activity")

	// 创建 Agent（需要 LLM 客户端和知识库）
	// llmClient := NewLLMClient(...)
	// knowledgeRepo := NewKnowledgeRepository(...)
	// troubleshooter := agent.NewTroubleshooter(llmClient, knowledgeRepo)

	// 桩实现
	time.Sleep(2 * time.Second)

	return &TroubleshooterResult{
		Confidence: 0.8,
		RootCause:  "Example root cause analysis",
		RemediationPlan: &RemediationPlan{
			Steps: []RemediationStep{
				{
					ID:          "fix1",
					Command:     "systemctl restart docker",
					Description: "Restart Docker daemon",
					Risk:        "low",
				},
			},
		},
	}, nil
}

// KnowledgeStorageActivity 知识存储活动
func KnowledgeStorageActivity(ctx context.Context, input KnowledgeStorageInput) error {
	log := logrus.WithContext(ctx)
	log.WithField("deployment_id", input.DeploymentID).Info("Executing knowledge storage activity")

	// 实际实现会将部署经验存储到 RAG 知识库
	// knowledgeRepo.Store(ctx, input)

	time.Sleep(1 * time.Second)
	return nil
}
