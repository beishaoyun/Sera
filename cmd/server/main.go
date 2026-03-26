package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/servermind/aixm/internal/api"
	"github.com/servermind/aixm/internal/config"
	"github.com/servermind/aixm/internal/database"
)

func main() {
	// 初始化日志
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	logrus.Info("Starting ServerMind API Server...")

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

	// 创建 API 服务器
	server, err := api.NewServer(cfg, db)
	if err != nil {
		logrus.Fatalf("Failed to create API server: %v", err)
	}

	// 优雅关闭
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 启动服务器
	go func() {
		logrus.Infof("Starting API server on port %s", cfg.Server.Port)
		if err := server.Start(); err != nil {
			logrus.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// 等待中断信号
	<-ctx.Done()

	// 优雅关闭
	logrus.Info("Shutting down API server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("Failed to shutdown server gracefully: %v", err)
	}

	logrus.Info("ServerMind API Server stopped")
}
