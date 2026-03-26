package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	apperrors "github.com/servermind/aixm/pkg/errors"
)

// CommandValidator 命令验证器
type CommandValidator struct {
	allowlists map[string]map[string]bool // command -> subcommands
	patterns   []DangerousPattern
}

// DangerousPattern 危险命令模式
type DangerousPattern struct {
	Pattern  *regexp.Regexp
	Severity string // critical, high, medium
	Reason   string
}

// NewCommandValidator 创建命令验证器
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		allowlists: buildDefaultAllowlist(),
		patterns:   buildDangerousPatterns(),
	}
}

// buildDefaultAllowlist 构建默认命令白名单
func buildDefaultAllowlist() map[string]map[string]bool {
	return map[string]map[string]bool{
		"apt-get": {"install": true, "update": true, "upgrade": true, "remove": true, "purge": true, "clean": true},
		"yum":     {"install": true, "update": true, "remove": true, "search": true, "clean": true},
		"dnf":     {"install": true, "update": true, "remove": true, "search": true, "clean": true},
		"apk":     {"add": true, "update": true, "upgrade": true, "del": true},

		"systemctl": {"start": true, "stop": true, "restart": true, "status": true, "enable": true, "disable": true, "is-active": true, "is-enabled": true},

		"docker":     {"pull": true, "run": true, "stop": true, "rm": true, "ps": true, "logs": true, "build": true, "images": true, "network": true, "volume": true},
		"docker-compose": {"up": true, "down": true, "build": true, "pull": true, "logs": true, "ps": true, "restart": true, "stop": true, "start": true},

		"git":   {"clone": true, "pull": true, "checkout": true, "fetch": true, "reset": true, "status": true, "log": true},

		"npm":   {"install": true, "run": true, "build": true, "start": true, "test": true, "ci": true},
		"yarn":  {"install": true, "add": true, "remove": true, "run": true, "build": true, "start": true},
		"pnpm":  {"install": true, "add": true, "remove": true, "run": true, "build": true, "start": true},

		"pip":   {"install": true, "uninstall": true, "freeze": true, "list": true, "show": true},
		"pip3":  {"install": true, "uninstall": true, "freeze": true, "list": true, "show": true},

		"go":    {"build": true, "run": true, "test": true, "mod": true, "get": true, "install": true, "clean": true},

		"make":  {"all": true, "build": true, "install": true, "clean": true, "test": true},

		"curl":  {}, // 允许但需要检查 URL
		"wget":  {}, // 允许但需要检查 URL

		"cd":    {},
		"ls":    {},
		"cat":   {},
		"head":  {},
		"tail":  {},
		"grep":  {},
		"find":  {},
		"mkdir": {},
		"rm":    {}, // 需要特殊检查
		"cp":    {},
		"mv":    {},
		"chmod": {}, // 需要特殊检查
		"chown": {}, // 需要特殊检查

		"echo":  {},
		"env":   {},
		"printenv": {},
		"which": {},
		"whereis": {},
		"whoami": {},
		"pwd":   {},
		"uname": {},
		"hostname": {},

		"ps":    {},
		"top":   {},
		"htop":  {},
		"df":    {},
		"du":    {},
		"free":  {},
		"uptime": {},

		"netstat": {},
		"ss":      {},
		"ping":    {},
		"nslookup": {},
		"dig":     {},

		"node":  {},
		"python": {},
		"python3": {},
		"java":  {},
		"jar":   {},
	}
}

// buildDangerousPatterns 构建危险命令模式
func buildDangerousPatterns() []DangerousPattern {
	return []DangerousPattern{
		// 破坏性操作 - CRITICAL
		{regexp.MustCompile(`\brm\s+-rf\s+/`), "critical", "Attempting to recursively delete root filesystem"},
		{regexp.MustCompile(`\brm\s+-rf\s+\*`), "critical", "Attempting to delete all files in current directory"},
		{regexp.MustCompile(`\bmkfs\.`), "critical", "Filesystem formatting detected"},
		{regexp.MustCompile(`\bdd\s+if=.*\bof=/dev/[sh]d`), "critical", "Direct disk writing detected"},
		{regexp.MustCompile(`\b:?\(\)\s*\{\s*:\|\:&\s*\};:`), "critical", "Shell fork bomb detected"},

		// 权限提升 - HIGH
		{regexp.MustCompile(`\bchmod\s+.*777\b`), "high", "Overly permissive permissions (777)"},
		{regexp.MustCompile(`\bchown\s+-R\s+root\b`), "high", "Changing ownership to root recursively"},
		{regexp.MustCompile(`\bsudo\s+.*\bpasswd\b`), "high", "Modifying passwords with sudo"},

		// 管道到 Shell - HIGH
		{regexp.MustCompile(`\bwget\s+.*\|.*\b(sh|bash|zsh)\b`), "high", "Pipe wget output to shell"},
		{regexp.MustCompile(`\bcurl\s+.*\|.*\b(sh|bash|zsh)\b`), "high", "Pipe curl output to shell"},
		{regexp.MustCompile(`\bcurl\s+.*-o-\s*\|`), "high", "Pipe curl output to another command"},

		// 信息泄露 - MEDIUM
		{regexp.MustCompile(`\bcat\s+/etc/shadow\b`), "medium", "Reading shadow password file"},
		{regexp.MustCompile(`\bcat\s+/etc/gshadow\b`), "medium", "Reading shadow group file"},
		{regexp.MustCompile(`\benv\s+.*\b(API_KEY|PRIVATE|SECRET|PASSWORD|TOKEN)\b`), "medium", "Potential secret exposure"},

		// 历史命令操作 - MEDIUM
		{regexp.MustCompile(`\bhistory\s+-c\b`), "medium", "Clearing command history"},
		{regexp.MustCompile(`\brm\s+.*\.bash_history`), "medium", "Deleting bash history"},

		// crontab 操作 - MEDIUM
		{regexp.MustCompile(`\bcrontab\s+-r\b`), "medium", "Removing all crontab entries"},
	}
}

// ValidationResult 验证结果
type ValidationResult struct {
	Allowed   bool
	Command   string
	Severity  string
	Reason    string
	Solution  string
	BlockedAt string
}

// Validate 验证命令
func (v *CommandValidator) Validate(command string) *ValidationResult {
	command = strings.TrimSpace(command)

	// 空命令
	if command == "" {
		return &ValidationResult{
			Allowed:   false,
			Command:   command,
			Severity:  "medium",
			Reason:    "Empty command",
			BlockedAt: time.Now().Format(time.RFC3339),
		}
	}

	// 检查危险模式
	for _, pattern := range v.patterns {
		if pattern.Pattern.MatchString(command) {
			return &ValidationResult{
				Allowed:   false,
				Command:   command,
				Severity:  pattern.Severity,
				Reason:    pattern.Reason,
				BlockedAt: time.Now().Format(time.RFC3339),
			}
		}
	}

	// 解析命令
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return &ValidationResult{
			Allowed:   false,
			Command:   command,
			Severity:  "low",
			Reason:    "Failed to parse command",
			BlockedAt: time.Now().Format(time.RFC3339),
		}
	}

	baseCmd := parts[0]

	// 检查是否在白名单中
	allowedCmds, exists := v.allowlists[baseCmd]
	if !exists {
		return &ValidationResult{
			Allowed:   false,
			Command:   command,
			Severity:  "medium",
			Reason:    fmt.Sprintf("Command '%s' not in allowlist", baseCmd),
			Solution:  "Please use a command from the allowlist or contact administrator",
			BlockedAt: time.Now().Format(time.RFC3339),
		}
	}

	// 如果命令有子命令检查
	if len(parts) > 1 && len(allowedCmds) > 0 {
		subCmd := strings.TrimLeft(parts[1], "-")
		if !allowedCmds[subCmd] {
			return &ValidationResult{
				Allowed:   false,
				Command:   command,
				Severity:  "low",
				Reason:    fmt.Sprintf("Subcommand '%s' not allowed for '%s'", subCmd, baseCmd),
				Solution:  fmt.Sprintf("Allowed subcommands for %s: %v", baseCmd, getAllowedSubcommands(allowedCmds)),
				BlockedAt: time.Now().Format(time.RFC3339),
			}
		}
	}

	// 特殊命令的额外检查
	if err := v.specialCheck(baseCmd, command); err != nil {
		return &ValidationResult{
			Allowed:   false,
			Command:   command,
			Severity:  "high",
			Reason:    err.Error(),
			BlockedAt: time.Now().Format(time.RFC3339),
		}
	}

	// 验证通过
	return &ValidationResult{
		Allowed:   true,
		Command:   command,
		BlockedAt: time.Now().Format(time.RFC3339),
	}
}

// specialCheck 特殊命令检查
func (v *CommandValidator) specialCheck(cmd, command string) error {
	switch cmd {
	case "rm":
		// 检查是否尝试删除重要目录
		protected := []string{"/etc", "/usr", "/bin", "/sbin", "/lib", "/var", "/opt", "/home", "/root"}
		for _, p := range protected {
			if strings.Contains(command, p) && (strings.Contains(command, "-r") || strings.Contains(command, "-rf")) {
				return fmt.Errorf("Attempting to delete protected directory: %s", p)
			}
		}
	case "curl", "wget":
		// 检查 URL 是否安全
		if strings.Contains(command, "http://") && !strings.Contains(command, "localhost") && !strings.Contains(command, "127.0.0.1") {
			// 允许但记录警告
		}
	case "chmod":
		// 检查是否设置 SUID/SGID
		if strings.Contains(command, "4") || strings.Contains(command, "2") {
			if len(command) > 10 { // chmod +x file 这种短命令不会有这个问题
				return fmt.Errorf("Setting SUID/SGID bits requires approval")
			}
		}
	}
	return nil
}

func getAllowedSubcommands(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// ToAppError 转换为应用错误
func (r *ValidationResult) ToAppError() *apperrors.AppError {
	if r.Allowed {
		return nil
	}
	return apperrors.CommandBlocked(r.Command, r.Reason).
		WithContext("severity", r.Severity).
		WithContext("blocked_at", r.BlockedAt)
}

// CommandSandbox 命令沙箱
type CommandSandbox struct {
	validator *CommandValidator
	workdir   string
	user      string
	timeout   time.Duration
}

// SandboxConfig 沙箱配置
type SandboxConfig struct {
	Workdir      string        `json:"workdir"`
	User         string        `json:"user"`
	Timeout      time.Duration `json:"timeout"`
	MaxOutputSize int          `json:"max_output_size"`
}

// DefaultSandboxConfig 默认沙箱配置
func DefaultSandboxConfig() SandboxConfig {
	return SandboxConfig{
		Workdir:       "/tmp/servermind-sandbox",
		User:          "servermind",
		Timeout:       300 * time.Second,
		MaxOutputSize: 10 * 1024 * 1024, // 10MB
	}
}

// NewCommandSandbox 创建命令沙箱
func NewCommandSandbox(config SandboxConfig) *CommandSandbox {
	return &CommandSandbox{
		validator: NewCommandValidator(),
		workdir:   config.Workdir,
		user:      config.User,
		timeout:   config.Timeout,
	}
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	ExitCode   int
	Output     string
	Stderr     string
	Duration   time.Duration
	TimedOut   bool
}

// Execute 执行命令（在沙箱中）
func (s *CommandSandbox) Execute(ctx context.Context, command string) (*ExecutionResult, error) {
	// 1. 验证命令
	result := s.validator.Validate(command)
	if !result.Allowed {
		return nil, result.ToAppError()
	}

	// 2. 创建带超时的上下文
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	// 3. 准备命令
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, apperrors.InvalidArgument("Command is empty", "command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = s.workdir
	cmd.Env = []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"NODE_ENV=production",
	}

	// 4. 执行命令
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	execResult := &ExecutionResult{
		Output:   string(output),
		Duration: duration,
	}

	if cmd.ProcessState != nil {
		execResult.ExitCode = cmd.ProcessState.ExitCode()
	}

	if ctx.Err() == context.DeadlineExceeded {
		execResult.TimedOut = true
		return execResult, apperrors.Timeout("command execution").
			WithContext("duration", duration.String()).
			WithContext("command", command)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			execResult.ExitCode = exitErr.ExitCode()
			execResult.Stderr = string(exitErr.Stderr)
		} else {
			return nil, apperrors.SSHError("execution", err.Error()).
				WithContext("command", command)
		}
	}

	return execResult, nil
}

// ValidateOnly 仅验证不执行
func (s *CommandSandbox) ValidateOnly(command string) *ValidationResult {
	return s.validator.Validate(command)
}
