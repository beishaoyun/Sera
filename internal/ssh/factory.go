package ssh

import (
	"context"
	"fmt"
	"time"

	"github.com/servermind/aixm/internal/sandbox"
	"github.com/servermind/aixm/pkg/models"
)

// SSHClientFactory SSH 客户端工厂实现
type SSHClientFactory struct {
	pool    *SSHPool
	config  *SSHConfig
	sandbox *sandbox.CommandSandbox
}

// NewSSHClientFactory 创建 SSH 客户端工厂
func NewSSHClientFactory(config *SSHConfig, sb *sandbox.CommandSandbox) *SSHClientFactory {
	return &SSHClientFactory{
		config:  config,
		sandbox: sb,
		// 连接池大小为 10
		pool: NewSSHPool(10, config, sb),
	}
}

// GetClient 获取 SSH 客户端
func (f *SSHClientFactory) GetClient(ctx context.Context, serverID string) (SSHClient, error) {
	// 这里需要从数据库或服务中获取服务器配置
	// 目前返回一个使用默认配置的客户端
	return f.pool.GetConnection(ctx, serverID)
}

// Close 关闭连接池
func (f *SSHClientFactory) Close() {
	if f.pool != nil {
		f.pool.CloseAll()
	}
}

// ServerConfigFetcher 服务器配置获取接口
type ServerConfigFetcher interface {
	GetServerConfig(ctx context.Context, serverID string) (*models.Server, error)
}

// DatabaseSSHClientFactory 从数据库获取服务器配置的 SSH 工厂
type DatabaseSSHClientFactory struct {
	fetcher ServerConfigFetcher
	pool    *SSHPool
	sandbox *sandbox.CommandSandbox
}

// NewDatabaseSSHClientFactory 创建从数据库获取配置的 SSH 工厂
func NewDatabaseSSHClientFactory(fetcher ServerConfigFetcher, sb *sandbox.CommandSandbox) *DatabaseSSHClientFactory {
	return &DatabaseSSHClientFactory{
		fetcher: fetcher,
		sandbox: sb,
		pool:    NewSSHPool(10, nil, sb),
	}
}

// GetClient 获取 SSH 客户端（从数据库读取配置）
func (f *DatabaseSSHClientFactory) GetClient(ctx context.Context, serverID string) (SSHClient, error) {
	// 从数据库获取服务器配置
	server, err := f.fetcher.GetServerConfig(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server config: %w", err)
	}

	// 构建 SSH 配置
	sshConfig := &SSHConfig{
		Host:      server.Host,
		Port:      server.Port,
		Username:  server.Username,
		Timeout:   30 * time.Second,
		KeepAlive: 60 * time.Second,
	}

	// 注意：实际项目中，私钥/密码应该从 Vault 动态获取
	// 这里只是示意，实际应该通过 CredentialID 从 Vault 获取
	// if server.CredentialID != "" {
	//     creds, err := vaultClient.GetSSHKey(ctx, server.CredentialID)
	//     if err != nil { ... }
	//     sshConfig.PrivateKey = creds.PrivateKey
	// }

	// 创建客户端
	client := NewSSHClient(sshConfig, f.sandbox)
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return client, nil
}

// Close 关闭连接池
func (f *DatabaseSSHClientFactory) Close() {
	if f.pool != nil {
		f.pool.CloseAll()
	}
}
