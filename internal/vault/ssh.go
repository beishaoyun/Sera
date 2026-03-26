package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

// Config Vault 配置
type Config struct {
	Address string
	Token   string
	// SSH 密钥引擎路径
	SSHEnginePath string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Address:       "http://localhost:8200",
		Token:         "servermind_dev_token",
		SSHEnginePath: "ssh",
	}
}

// Client Vault 客户端
type Client struct {
	client        *api.Client
	config        *Config
	sshEnginePath string
}

// SSHKeyPair SSH 密钥对
type SSHKeyPair struct {
	PublicKey  string
	PrivateKey string
}

// SSHCertificate SSH 证书
type SSHCertificate struct {
	KeyID       string
	PublicKey   string
	PrivateKey  string
	ValidUntil  time.Time
	Principals  []string
	Extensions  map[string]string
}

// NewClient 创建 Vault 客户端
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	client, err := api.NewClient(&api.Config{
		Address: config.Address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(config.Token)

	c := &Client{
		client:        client,
		config:        config,
		sshEnginePath: config.SSHEnginePath,
	}

	// 健康检查
	if _, err := c.Sys().Health(); err != nil {
		logrus.Warnf("Vault health check failed: %v", err)
	}

	return c, nil
}

// InitSSHEngine 初始化 SSH 密钥引擎
func (c *Client) InitSSHEngine(ctx context.Context) error {
	// 启用 SSH 密钥引擎（如果未启用）
	mounts, err := c.client.Sys().ListMounts()
	if err != nil {
		return err
	}

	if _, exists := mounts[c.sshEnginePath+"/"]; !exists {
		// 启用 SSH 引擎
		if err := c.client.Sys().Mount(c.sshEnginePath, &api.MountInput{
			Type:        "ssh",
			Description: "SSH Key Management for Sera",
		}); err != nil {
			return fmt.Errorf("failed to mount SSH engine: %w", err)
		}
		logrus.Infof("Mounted SSH engine at %s", c.sshEnginePath)
	}

	// 配置 CA
	_, err = c.client.Logical().Write(c.sshEnginePath+"/config/ca", map[string]interface{}{
		"generate_signing_key": true,
		"key_type":             "ca",
		"key_bits":             4096,
	})
	if err != nil {
		return fmt.Errorf("failed to configure SSH CA: %w", err)
	}

	logrus.Info("SSH CA configured")
	return nil
}

// GenerateSSHKey 生成 SSH 密钥对
func (c *Client) GenerateSSHKey(ctx context.Context, keyName string, keyType string) (*SSHKeyPair, error) {
	if keyType == "" {
		keyType = "ed25519"
	}

	// 生成密钥
	secret, err := c.client.Logical().Write(c.sshEnginePath+"/key/"+keyName, map[string]interface{}{
		"key_type": keyType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate SSH key: %w", err)
	}

	// 获取公钥
	publicKeySecret, err := c.client.Logical().Read(c.sshEnginePath + "/key/" + keyName)
	if err != nil {
		return nil, err
	}

	publicKey, _ := publicKeySecret.Data["public_key"].(string)

	return &SSHKeyPair{
		PublicKey:  publicKey,
		PrivateKey: "", // 私钥不存储在 Vault 中
	}, nil
}

// SignSSHKey 签署 SSH 证书
func (c *Client) SignSSHKey(ctx context.Context, req *SignSSHKeyRequest) (*SSHCertificate, error) {
	// 准备签署请求
	data := map[string]interface{}{
		"public_key":       req.PublicKey,
		"key_id":           req.KeyID,
		"valid_principals": req.Principals,
		"ttl":              req.TTL.String(),
	}

	if len(req.Extensions) > 0 {
		data["extensions"] = req.Extensions
	}

	if len(req.CriticalOptions) > 0 {
		data["critical_options"] = req.CriticalOptions
	}

	// 签署证书
	secret, err := c.client.Logical().Write(c.sshEnginePath+"/sign/"+req.Role, data)
	if err != nil {
		return nil, fmt.Errorf("failed to sign SSH key: %w", err)
	}

	signedKey, ok := secret.Data["signed_key"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid signed key response")
	}

	// 解析证书
	cert, err := parseSSHCertificate(signedKey)
	if err != nil {
		return nil, err
	}

	cert.KeyID = req.KeyID

	logrus.WithFields(logrus.Fields{
		"key_id":     req.KeyID,
		"validUntil": cert.ValidUntil,
		"principals": req.Principals,
	}).Info("SSH certificate signed")

	return cert, nil
}

// SignSSHKeyRequest 签署 SSH 密钥请求
type SignSSHKeyRequest struct {
	KeyID          string            // 密钥 ID（通常是用户名）
	PublicKey      string            // 要签署的公钥
	Principals     []string          // 允许的主体（通常是用户名）
	TTL            time.Duration     // 有效期
	Extensions     map[string]string // 扩展
	CriticalOptions map[string]string // 关键选项
	Role           string            // 角色名称
}

// CreateRole 创建 SSH 签署角色
func (c *Client) CreateRole(ctx context.Context, name string, config SSHRoleConfig) error {
	data := map[string]interface{}{
		"key_type":                  config.KeyType,
		"allow_user_certificates":   config.AllowUserCertificates,
		"allow_host_certificates":   config.AllowHostCertificates,
		"allowed_users":             config.AllowedUsers,
		"allowed_domains":           config.AllowedDomains,
		"default_extensions":        config.DefaultExtensions,
		"default_critical_options":  config.DefaultCriticalOptions,
		"key_id_format":             config.KeyIDFormat,
		"max_ttl":                   config.MaxTTL.String(),
		"default_ttl":               config.DefaultTTL.String(),
		"allow_subdomains":          config.AllowSubdomains,
		"allow_bare_domains":        config.AllowBareDomains,
		"allow_user_key_ids":        config.AllowUserKeyIDs,
	}

	_, err := c.client.Logical().Write(c.sshEnginePath+"/roles/"+name, data)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	logrus.WithField("role", name).Info("SSH role created")
	return nil
}

// SSHRoleConfig SSH 角色配置
type SSHRoleConfig struct {
	KeyType                 string            // user, host
	AllowUserCertificates   bool              // 允许用户证书
	AllowHostCertificates   bool              // 允许主机证书
	AllowedUsers            []string          // 允许的用户
	AllowedDomains          []string          // 允许的域名
	DefaultExtensions       map[string]string // 默认扩展
	DefaultCriticalOptions  map[string]string // 默认关键选项
	KeyIDFormat             string            // KeyID 格式
	MaxTTL                  time.Duration     // 最大有效期
	DefaultTTL              time.Duration     // 默认有效期
	AllowSubdomains         bool              // 允许子域名
	AllowBareDomains        bool              // 允许裸域名
	AllowUserKeyIDs         bool              // 允许用户自定义 KeyID
}

// DefaultUserRole 创建默认用户角色
func (c *Client) DefaultUserRole(ctx context.Context) error {
	return c.CreateRole(ctx, "deploy-user", SSHRoleConfig{
		KeyType:               "user",
		AllowUserCertificates: true,
		AllowedUsers:          []string{"{{key_id}}"},
		DefaultExtensions: map[string]string{
			"permit-X11-forwarding":   "",
			"permit-agent-forwarding": "",
			"permit-port-forwarding":  "",
			"permit-pty":              "",
			"permit-user-rc":          "",
		},
		DefaultTTL:    24 * time.Hour,
		MaxTTL:        7 * 24 * time.Hour,
		AllowBareDomains: true,
	})
}

// GetSSHCertificate 获取 SSH 证书（从存储中）
func (c *Client) GetSSHCertificate(ctx context.Context, deploymentID string) (*SSHCertificate, error) {
	// 从 Vault 的 KV 存储中获取证书
	secret, err := c.client.Logical().Read("secret/data/sera/ssh/" + deploymentID)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("certificate not found")
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid certificate data")
	}

	privateKey, _ := data["private_key"].(string)
	publicKey, _ := data["public_key"].(string)
	keyID, _ := data["key_id"].(string)
	validUntilStr, _ := data["valid_until"].(string)

	validUntil, _ := time.Parse(time.RFC3339, validUntilStr)

	return &SSHCertificate{
		KeyID:      keyID,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		ValidUntil: validUntil,
	}, nil
}

// StoreSSHCertificate 存储 SSH 证书
func (c *Client) StoreSSHCertificate(ctx context.Context, deploymentID string, cert *SSHCertificate) error {
	_, err := c.client.Logical().Write("secret/data/sera/ssh/"+deploymentID, map[string]interface{}{
		"key_id":      cert.KeyID,
		"public_key":  cert.PublicKey,
		"private_key": cert.PrivateKey,
		"valid_until": cert.ValidUntil.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("failed to store certificate: %w", err)
	}

	logrus.WithField("deployment_id", deploymentID).Info("SSH certificate stored")
	return nil
}

// DeleteSSHCertificate 删除 SSH 证书
func (c *Client) DeleteSSHCertificate(ctx context.Context, deploymentID string) error {
	_, err := c.client.Logical().Delete("secret/data/sera/ssh/" + deploymentID)
	return err
}

// parseSSHCertificate 解析 SSH 证书
func parseSSHCertificate(signedKey string) (*SSHCertificate, error) {
	// 简单解析 SSH 证书
	// 实际实现应该使用 golang.org/x/crypto/ssh 包解析
	return &SSHCertificate{
		PublicKey: signedKey,
	}, nil
}

// Close 关闭连接
func (c *Client) Close() {
	// Vault 客户端不需要显式关闭
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	health, err := c.client.Sys().Health()
	if err != nil {
		return err
	}

	if health.StatusCode != 200 {
		return fmt.Errorf("vault unhealthy: status %d", health.StatusCode)
	}

	return nil
}
