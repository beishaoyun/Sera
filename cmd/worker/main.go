package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/servermind/aixm/internal/config"
	"github.com/servermind/aixm/internal/agent"
	"github.com/servermind/aixm/internal/database"
	"github.com/servermind/aixm/internal/ssh"
)

func main() {
	// 初始化日志
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	logrus.Info("Starting ServerMind Worker...")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := database.NewDatabase(cfg.Database)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	logrus.Info("Database connection established")

	// 初始化 SSH 连接池
	sshConfig := &ssh.SSHConfig{
		Timeout:   cfg.SSH.ConnectTimeout,
		KeepAlive: cfg.SSH.KeepAlive,
	}
	sshPool := ssh.NewSSHPool(cfg.SSH.MaxConnections, sshConfig)

	// 初始化 Agent（实际项目中需要实现 LLM 客户端和知识库）
	// llmClient := NewLLMClient(cfg.LLM)
	// knowledgeRepo := NewKnowledgeRepository(cfg.Milvus)

	// requirementParser := agent.NewRequirementParser(llmClient)
	// codeAnalyzer := agent.NewCodeAnalyzer(llmClient)
	// deploymentExecutor := agent.NewDeploymentExecutor(sshPool)
	// troubleshooter := agent.NewTroubleshooter(llmClient, knowledgeRepo)

	// 初始化 Temporal Worker（用于部署工作流）
	// temporalClient, err := client.Dial(client.Options{
	// 	HostPort: cfg.Temporal.HostPort,
	// 	Namespace: cfg.Temporal.Namespace,
	// })
	// if err != nil {
	// 	logrus.Fatalf("Failed to connect to Temporal: %v", err)
	// }
	// defer temporalClient.Close()

	// worker := worker.New(temporalClient, "deployment-queue")
	// worker.RegisterWorkflow(DeploymentWorkflow)
	// worker.RegisterActivity(...)

	// 优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logrus.Info("Worker is ready to process deployment tasks")

	// 启动 Worker 循环（简化版本，实际应使用 Temporal）
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 从队列获取任务并处理
				// task, err := taskQueue.Dequeue(ctx)
				// if err != nil {
				// 	time.Sleep(1 * time.Second)
				// 	continue
				// }
				// processTask(ctx, task)
				time.Sleep(5 * time.Second)
			}
		}
	}()

	// 等待中断信号
	<-ctx.Done()

	// 优雅关闭
	logrus.Info("Shutting down Worker...")
	sshPool.CloseAll()

	logrus.Info("ServerMind Worker stopped")
}
