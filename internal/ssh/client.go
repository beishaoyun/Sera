package ssh

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// SSHClient SSH 客户端接口
type SSHClient interface {
	// Connect 建立连接
	Connect(ctx context.Context) error
	// Execute 执行命令
	Execute(ctx context.Context, command string, opts ExecuteOptions) (*ExecuteResult, error)
	// ExecuteStream 流式执行（实时返回输出）
	ExecuteStream(ctx context.Context, command string, outputHandler func(string)) error
	// Upload 上传文件
	Upload(ctx context.Context, localPath, remotePath string) error
	// Download 下载文件
	Download(ctx context.Context, remotePath, localPath string) error
	// Close 关闭连接
	Close() error
	// IsConnected 检查连接状态
	IsConnected() bool
}

// ExecuteOptions 执行选项
type ExecuteOptions struct {
	Timeout         time.Duration
	WorkingDir      string
	Environment     map[string]string
	User            string
	MaxOutputSize   int // 最大输出字节数
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	ExitCode   int
	Output     string
	Error      error
	Duration   time.Duration
	Timestamp  time.Time
}

// SSHConfig SSH 连接配置
type SSHConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	PrivateKey     []byte
	Timeout        time.Duration
	KeepAlive      time.Duration
	HostKeyCallback ssh.HostKeyCallback
}

// sshClient SSH 客户端实现
type sshClient struct {
	config     *SSHConfig
	client     *ssh.Client
	mu         sync.RWMutex
	lastUsed   time.Time
	closed     bool
}

// NewSSHClient 创建 SSH 客户端
func NewSSHClient(config *SSHConfig) SSHClient {
	return &sshClient{
		config:   config,
		lastUsed: time.Now(),
	}
}

func (s *sshClient) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return nil // 已连接
	}

	// 构建 SSH 配置
	sshConfig := &ssh.ClientConfig{
		User: s.config.Username,
		Auth: []ssh.AuthMethod{},
		Timeout: s.config.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该使用 proper host key verification
	}

	// 添加认证方法
	if s.config.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(s.config.Password))
	}
	if len(s.config.PrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(s.config.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial SSH: %w", err)
	}

	s.client = client
	s.lastUsed = time.Now()
	s.closed = false

	logrus.WithFields(logrus.Fields{
		"host": s.config.Host,
		"user": s.config.Username,
	}).Info("SSH connection established")

	return nil
}

func (s *sshClient) Execute(ctx context.Context, command string, opts ExecuteOptions) (*ExecuteResult, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("SSH client not connected")
	}

	startTime := time.Now()

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// 设置超时
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// 设置环境变量
	for key, value := range opts.Environment {
		if err := session.Setenv(key, value); err != nil {
			logrus.Warnf("Failed to set env %s: %v", key, err)
		}
	}

	// 设置工作目录
	if opts.WorkingDir != "" {
		command = fmt.Sprintf("cd %s && %s", opts.WorkingDir, command)
	}

	// 执行命令并捕获输出
	var output []byte
	if err != nil {
		output, err = session.CombinedOutput(command)
	} else {
		output, err = session.Output(command)
	}

	result := &ExecuteResult{
		Output:    string(output),
		Duration:  time.Since(startTime),
		Timestamp: startTime,
	}

	// 处理错误
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitErr.ExitStatus()
			result.Error = fmt.Errorf("command exited with code %d: %s", exitErr.ExitStatus(), string(output))
		} else {
			result.Error = err
		}
	}

	return result, nil
}

func (s *sshClient) ExecuteStream(ctx context.Context, command string, outputHandler func(string)) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("SSH client not connected")
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// 创建管道捕获输出
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// 启动命令
	if err := session.Start(command); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// 异步读取输出
	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stdout.Read(buf)
				if n > 0 {
					outputHandler(string(buf[:n]))
				}
				if err != nil {
					if err != io.EOF {
						logrus.Errorf("Error reading stdout: %v", err)
					}
					break
				}
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stderr.Read(buf)
				if n > 0 {
					outputHandler(string(buf[:n]))
				}
				if err != nil {
					if err != io.EOF {
						logrus.Errorf("Error reading stderr: %v", err)
					}
					break
				}
			}
		}
	}()

	// 等待命令完成
	if err := session.Wait(); err != nil {
		return err
	}

	return nil
}

func (s *sshClient) Upload(ctx context.Context, localPath, remotePath string) error {
	// TODO: 实现 SFTP 上传
	return fmt.Errorf("not implemented")
}

func (s *sshClient) Download(ctx context.Context, remotePath, localPath string) error {
	// TODO: 实现 SFTP 下载
	return fmt.Errorf("not implemented")
}

func (s *sshClient) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil && !s.closed {
		if err := s.client.Close(); err != nil {
			return err
		}
		s.client = nil
		s.closed = true
		logrus.Info("SSH connection closed")
	}
	return nil
}

func (s *sshClient) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client != nil && !s.closed
}

// SSHPool SSH 连接池
type SSHPool struct {
	connections map[string]SSHClient
	mu          sync.RWMutex
	maxSize     int
	config      *SSHConfig
}

// NewSSHPool 创建 SSH 连接池
func NewSSHPool(maxSize int, config *SSHConfig) *SSHPool {
	return &SSHPool{
		connections: make(map[string]SSHClient),
		maxSize:     maxSize,
		config:      config,
	}
}

func (p *SSHPool) GetConnection(ctx context.Context, serverID string) (SSHClient, error) {
	p.mu.RLock()
	client, exists := p.connections[serverID]
	p.mu.RUnlock()

	if exists && client.IsConnected() {
		return client, nil
	}

	// 创建新连接
	client = NewSSHClient(p.config)
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.connections[serverID] = client
	p.mu.Unlock()

	return client, nil
}

func (p *SSHPool) ReleaseConnection(serverID string) {
	p.mu.RLock()
	client, exists := p.connections[serverID]
	p.mu.RUnlock()

	if exists {
		client.Close()
		p.mu.Lock()
		delete(p.connections, serverID)
		p.mu.Unlock()
	}
}

func (p *SSHPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for serverID, client := range p.connections {
		client.Close()
		delete(p.connections, serverID)
	}
}
