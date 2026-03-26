package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Client GitHub API 客户端
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	rateLimit  *RateLimit
}

// RateLimit 速率限制信息
type RateLimit struct {
	Remaining int
	Limit     int
	Reset     time.Time
}

// Config 客户端配置
type Config struct {
	APIKey  string
	BaseURL string // 可选，用于 GitHub Enterprise
}

// NewClient 创建 GitHub API 客户端
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.github.com"
	}

	return &Client{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithToken 使用 Token 创建客户端
func NewClientWithToken(token string) *Client {
	return NewClient(Config{APIKey: token})
}

// RepoInfo 仓库信息
type RepoInfo struct {
	Owner       string
	Repo        string
	Branch      string
	FullPath    string
}

// Repository 仓库元数据
type Repository struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Homepage      string `json:"homepage"`
	Language      string `json:"language"`
	Stars         int    `json:"stargazers_count"`
	Forks         int    `json:"forks_count"`
	DefaultBranch string `json:"default_branch"`
	UpdatedAt     string `json:"updated_at"`
	HasWiki       bool   `json:"has_wiki"`
	HasPages      bool   `json:"has_pages"`
}

// FileEntry 文件条目
type FileEntry struct {
	Path        string `json:"path"`
	Type        string `json:"type"` // file, dir
	Name        string `json:"name"`
	Size        int    `json:"size"`
	DownloadURL string `json:"download_url"`
	GitURL      string `json:"git_url"`
	SHA         string `json:"sha"`
}

// FileContent 文件内容
type FileContent struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Encoding    string `json:"encoding"`
	Content     string `json:"content"` // Base64 编码
	Decoded     string // 解码后的内容
	Size        int    `json:"size"`
	DownloadURL string `json:"download_url"`
	SHA         string `json:"sha"`
}

// ParseURL 解析 GitHub URL
func ParseURL(rawURL string) (*RepoInfo, error) {
	// 支持格式:
	// https://github.com/owner/repo
	// https://github.com/owner/repo/tree/branch
	// https://github.com/owner/repo/tree/branch/path
	// git@github.com:owner/repo.git

	rawURL = strings.TrimSuffix(rawURL, ".git")
	rawURL = strings.TrimSuffix(rawURL, "/")

	// SSH 格式
	if strings.HasPrefix(rawURL, "git@github.com:") {
		parts := strings.TrimPrefix(rawURL, "git@github.com:")
		parts = strings.Split(parts, "/")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid SSH GitHub URL")
		}
		return &RepoInfo{
			Owner:  parts[0],
			Repo:   parts[1],
			Branch: "main",
		}, nil
	}

	// HTTPS 格式
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Host != "github.com" {
		return nil, fmt.Errorf("not a GitHub URL")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL: missing owner/repo")
	}

	info := &RepoInfo{
		Owner:  parts[0],
		Repo:   parts[1],
		Branch: "main", // 默认
	}

	// 解析 branch 和 path
	if len(parts) >= 4 && parts[2] == "tree" {
		info.Branch = parts[3]
	}

	return info, nil
}

// GetRepository 获取仓库元数据
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var repoData Repository
	if err := json.NewDecoder(resp.Body).Decode(&repoData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"repo":  repoData.FullName,
		"stars": repoData.Stars,
	}).Info("GitHub repository fetched")

	return &repoData, nil
}

// GetFileTree 获取文件树
func (c *Client) GetFileTree(ctx context.Context, owner, repo, branch string) ([]FileEntry, error) {
	// 使用 Git Trees API
	endpoint := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1", c.baseURL, owner, repo, branch)

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var treeResp struct {
		Tree []FileEntry `json:"tree"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 过滤掉过深的文件（避免 token 浪费）
	var filtered []FileEntry
	for _, entry := range treeResp.Tree {
		// 只获取文件和必要的目录
		if entry.Type == "blob" || strings.Count(entry.Path, "/") <= 1 {
			filtered = append(filtered, entry)
		}
	}

	logrus.WithFields(logrus.Fields{
		"files": len(filtered),
	}).Info("GitHub file tree fetched")

	return filtered, nil
}

// GetREADME 获取 README 内容
func (c *Client) GetREADME(ctx context.Context, owner, repo, branch string) (string, error) {
	// 尝试多种 README 文件名
	readmeNames := []string{"README.md", "README.rst", "README.txt", "README"}

	for _, name := range readmeNames {
		content, err := c.GetFile(ctx, owner, repo, name, branch)
		if err == nil {
			return content.Decoded, nil
		}
	}

	// 尝试从 API 直接获取
	endpoint := fmt.Sprintf("%s/repos/%s/%s/readme?ref=%s", c.baseURL, owner, repo, branch)
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("README not found: %w", err)
	}
	defer resp.Body.Close()

	var fileResp FileContent
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return "", err
	}

	// Base64 解码
	decoded, err := base64.StdEncoding.DecodeString(fileResp.Content)
	if err != nil {
		return "", err
	}
	fileResp.Decoded = string(decoded)

	return fileResp.Decoded, nil
}

// GetFile 获取单个文件内容
func (c *Client) GetFile(ctx context.Context, owner, repo, path, branch string) (*FileContent, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", c.baseURL, owner, repo, path, branch)

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var fileResp FileContent
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return nil, err
	}

	// Base64 解码
	if fileResp.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(fileResp.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode content: %w", err)
		}
		fileResp.Decoded = string(decoded)
	}

	return &fileResp, nil
}

// GetRawFile 获取原始文件内容（直接返回文本）
func (c *Client) GetRawFile(ctx context.Context, owner, repo, path, branch string) (string, error) {
	endpoint := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, path)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file not found: %s", path)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// GetRateLimit 获取速率限制状态
func (c *Client) GetRateLimit(ctx context.Context) (*RateLimit, error) {
	endpoint := fmt.Sprintf("%s/rate_limit", c.baseURL)

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rateResp struct {
		Resources struct {
			Core struct {
				Limit     int `json:"limit"`
				Remaining int `json:"remaining"`
				Reset     int `json:"reset"`
			} `json:"core"`
		} `json:"resources"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rateResp); err != nil {
		return nil, err
	}

	c.rateLimit = &RateLimit{
		Limit:     rateResp.Resources.Core.Limit,
		Remaining: rateResp.Resources.Core.Remaining,
		Reset:     time.Unix(int64(rateResp.Resources.Core.Reset), 0),
	}

	return c.rateLimit, nil
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	// 设置认证头
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// GitHub API 要求 User-Agent
	req.Header.Set("User-Agent", "Sera-Platform/1.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// 检查速率限制
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		logrus.Debugf("GitHub API rate limit remaining: %s", remaining)
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}

// FetchRepoMetadata 获取仓库完整元数据
func (c *Client) FetchRepoMetadata(ctx context.Context, repoURL string) (*Repository, *RepoInfo, error) {
	info, err := ParseURL(repoURL)
	if err != nil {
		return nil, nil, err
	}

	repo, err := c.GetRepository(ctx, info.Owner, info.Repo)
	if err != nil {
		return nil, nil, err
	}

	return repo, info, nil
}

// FetchRepoAnalysis 获取仓库分析所需的全部数据
func (c *Client) FetchRepoAnalysis(ctx context.Context, repoURL string) (*Repository, []FileEntry, string, error) {
	repo, info, err := c.FetchRepoMetadata(ctx, repoURL)
	if err != nil {
		return nil, nil, "", err
	}

	fileTree, err := c.GetFileTree(ctx, info.Owner, info.Repo, info.Branch)
	if err != nil {
		return nil, nil, "", err
	}

	readme, err := c.GetREADME(ctx, info.Owner, info.Repo, info.Branch)
	if err != nil {
		logrus.Warnf("Failed to fetch README: %v", err)
		// README 不是必须的，继续执行
	}

	return repo, fileTree, readme, nil
}
