package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorCode 错误码
type ErrorCode string

const (
	// 通用错误
	ErrUnknown         ErrorCode = "UNKNOWN"
	ErrInvalidArgument ErrorCode = "INVALID_ARGUMENT"
	ErrNotFound        ErrorCode = "NOT_FOUND"
	ErrAlreadyExists   ErrorCode = "ALREADY_EXISTS"
	ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"

	// 认证/授权错误
	ErrUnauthenticated ErrorCode = "UNAUTHENTICATED"
	ErrUnauthorized    ErrorCode = "UNAUTHORIZED"

	// 系统错误
	ErrInternal      ErrorCode = "INTERNAL"
	ErrUnavailable   ErrorCode = "UNAVAILABLE"
	ErrTimeout       ErrorCode = "TIMEOUT"
	ErrCancelled     ErrorCode = "CANCELLED"

	// Agent 相关错误
	ErrAgentNotFound     ErrorCode = "AGENT_NOT_FOUND"
	ErrAgentExecution    ErrorCode = "AGENT_EXECUTION"
	ErrAgentTimeout      ErrorCode = "AGENT_TIMEOUT"
	ErrAgentInvalidInput ErrorCode = "AGENT_INVALID_INPUT"

	// NATS 相关错误
	ErrNATSConnection  ErrorCode = "NATS_CONNECTION"
	ErrNATSPublish     ErrorCode = "NATS_PUBLISH"
	ErrNATSSubscribe   ErrorCode = "NATS_SUBSCRIBE"
	ErrNATSRequest     ErrorCode = "NATS_REQUEST"
	ErrNATSNoResponder ErrorCode = "NATS_NO_RESPONDER"

	// 部署相关错误
	ErrDeploymentFailed    ErrorCode = "DEPLOYMENT_FAILED"
	ErrDeploymentTimeout   ErrorCode = "DEPLOYMENT_TIMEOUT"
	ErrDeploymentCancelled ErrorCode = "DEPLOYMENT_CANCELLED"
	ErrDeploymentRollback  ErrorCode = "DEPLOYMENT_ROLLBACK"

	// SSH 相关错误
	ErrSSHConnection   ErrorCode = "SSH_CONNECTION"
	ErrSSHExecution    ErrorCode = "SSH_EXECUTION"
	ErrSSHTimeout      ErrorCode = "SSH_TIMEOUT"
	ErrSSHPermission   ErrorCode = "SSH_PERMISSION"

	// RAG 相关错误
	ErrRAGSearch       ErrorCode = "RAG_SEARCH"
	ErrRAGNotFound     ErrorCode = "RAG_NOT_FOUND"

	// 云厂商相关错误
	ErrCloudProvider   ErrorCode = "CLOUD_PROVIDER"
	ErrCloudQuota      ErrorCode = "CLOUD_QUOTA"
	ErrCloudBilling    ErrorCode = "CLOUD_BILLING"

	// 命令安全检查错误
	ErrCommandBlocked  ErrorCode = "COMMAND_BLOCKED"
	ErrCommandInvalid  ErrorCode = "COMMAND_INVALID"
	ErrSandboxEscape   ErrorCode = "SANDBOX_ESCAPE"
)

// Severity 错误严重程度
type Severity string

const (
	SeverityDebug    Severity = "debug"
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// AppError 统一应用错误
type AppError struct {
	// 错误码
	Code ErrorCode `json:"code"`

	// 错误消息（对用户友好）
	Message string `json:"message"`

	// 原始错误
	Err error `json:"-"`

	// 错误严重程度
	Severity Severity `json:"severity"`

	// 发生错误的步骤/阶段
	Step string `json:"step,omitempty"`

	// 上下文信息
	Context map[string]interface{} `json:"context,omitempty"`

	// 是否是可重试错误
	Retryable bool `json:"retryable"`

	// 建议的解决方案
	Solution string `json:"solution,omitempty"`

	// 堆栈跟踪
	Stack []StackFrame `json:"stack,omitempty"`

	// 时间戳
	Timestamp time.Time `json:"timestamp"`

	// 关联 ID（用于追踪）
	CorrelationID string `json:"correlation_id,omitempty"`

	// Agent 名称（如果是 Agent 错误）
	AgentName string `json:"agent_name,omitempty"`
}

// StackFrame 堆栈帧
type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

// New 创建新错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:      code,
		Message:   message,
		Severity:  SeverityError,
		Retryable: false,
		Timestamp: time.Now(),
		Stack:     captureStack(),
	}
}

// Newf 创建带格式的错误
func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:      code,
		Message:   fmt.Sprintf(format, args...),
		Severity:  SeverityError,
		Retryable: false,
		Timestamp: time.Now(),
		Stack:     captureStack(),
	}
}

// Wrap 包装现有错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:      code,
		Message:   message,
		Err:       err,
		Severity:  SeverityError,
		Retryable: false,
		Timestamp: time.Now(),
		Stack:     captureStack(),
	}
}

// Wrapf 包装现有错误（带格式）
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:      code,
		Message:   fmt.Sprintf(format, args...),
		Err:       err,
		Severity:  SeverityError,
		Retryable: false,
		Timestamp: time.Now(),
		Stack:     captureStack(),
	}
}

// WithSeverity 设置严重程度
func (e *AppError) WithSeverity(severity Severity) *AppError {
	e.Severity = severity
	return e
}

// WithStep 设置步骤
func (e *AppError) WithStep(step string) *AppError {
	e.Step = step
	return e
}

// WithContext 添加上下文
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithContextMap 批量添加上下文
func (e *AppError) WithContextMap(ctx map[string]interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	for k, v := range ctx {
		e.Context[k] = v
	}
	return e
}

// WithRetryable 设置是否可重试
func (e *AppError) WithRetryable(retryable bool) *AppError {
	e.Retryable = retryable
	return e
}

// WithSolution 设置解决方案
func (e *AppError) WithSolution(solution string) *AppError {
	e.Solution = solution
	return e
}

// WithCorrelationID 设置关联 ID
func (e *AppError) WithCorrelationID(id string) *AppError {
	e.CorrelationID = id
	return e
}

// WithAgentName 设置 Agent 名称
func (e *AppError) WithAgentName(name string) *AppError {
	e.AgentName = name
	return e
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// Is 实现 errors.Is 接口
func (e *AppError) Is(target error) bool {
	if appErr, ok := target.(*AppError); ok {
		return e.Code == appErr.Code
	}
	return false
}

// ToMap 转换为 map
func (e *AppError) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"code":       string(e.Code),
		"message":    e.Message,
		"severity":   string(e.Severity),
		"retryable":  e.Retryable,
		"timestamp":  e.Timestamp.Format(time.RFC3339),
	}

	if e.Step != "" {
		m["step"] = e.Step
	}
	if e.Context != nil {
		m["context"] = e.Context
	}
	if e.Solution != "" {
		m["solution"] = e.Solution
	}
	if e.CorrelationID != "" {
		m["correlation_id"] = e.CorrelationID
	}
	if e.AgentName != "" {
		m["agent_name"] = e.AgentName
	}
	if e.Stack != nil {
		m["stack"] = e.Stack
	}

	return m
}

// captureStack 捕获堆栈跟踪
func captureStack() []StackFrame {
	var frames []StackFrame
	buf := make([]uintptr, 10)
	n := runtime.Callers(2, buf)

	if n == 0 {
		return nil
	}

	runtimeFrames := runtime.CallersFrames(buf[:n])
	for {
		frame, more := runtimeFrames.Next()
		frames = append(frames, StackFrame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		})
		if !more {
			break
		}
	}

	return frames
}

// ============================================================================
// 便捷错误创建函数
// ============================================================================

// InvalidArgument 无效参数错误
func InvalidArgument(message string, field string) *AppError {
	return New(ErrInvalidArgument, message).
		WithSeverity(SeverityWarning).
		WithContext("field", field)
}

// NotFound 资源不存在错误
func NotFound(resourceType, id string) *AppError {
	return New(ErrNotFound, fmt.Sprintf("%s not found: %s", resourceType, id)).
		WithSeverity(SeverityWarning).
		WithContext("resource_type", resourceType).
		WithContext("resource_id", id)
}

// Internal 内部错误
func Internal(message string) *AppError {
	return New(ErrInternal, message).
		WithSeverity(SeverityError)
}

// Timeout 超时错误
func Timeout(operation string) *AppError {
	return Newf(ErrTimeout, "operation timed out: %s", operation).
		WithSeverity(SeverityError).
		WithRetryable(true)
}

// AgentError Agent 执行错误
func AgentError(agentName, message string) *AppError {
	return New(ErrAgentExecution, message).
		WithAgentName(agentName).
		WithSeverity(SeverityError)
}

// NATSError NATS 操作错误
func NATSError(operation, message string) *AppError {
	var code ErrorCode
	switch operation {
	case "publish":
		code = ErrNATSPublish
	case "subscribe":
		code = ErrNATSSubscribe
	case "request":
		code = ErrNATSRequest
	case "connection":
		code = ErrNATSConnection
	case "no_responder":
		code = ErrNATSNoResponder
		return New(code, message).
			WithSeverity(SeverityError).
			WithRetryable(true)
	default:
		code = ErrNATSConnection
	}

	return New(code, message).
		WithSeverity(SeverityError)
}

// DeploymentError 部署错误
func DeploymentError(message string, step string, retryable bool) *AppError {
	return New(ErrDeploymentFailed, message).
		WithStep(step).
		WithRetryable(retryable).
		WithSeverity(SeverityError)
}

// SSHError SSH 操作错误
func SSHError(operation, message string) *AppError {
	var code ErrorCode
	switch operation {
	case "connection":
		code = ErrSSHConnection
	case "execution":
		code = ErrSSHExecution
	case "timeout":
		code = ErrSSHTimeout
		return New(code, message).
			WithSeverity(SeverityError).
			WithRetryable(true)
	case "permission":
		code = ErrSSHPermission
	default:
		code = ErrSSHExecution
	}

	return New(code, message).
		WithSeverity(SeverityError)
}

// CommandBlocked 命令被阻止（安全检查失败）
func CommandBlocked(command, reason string) *AppError {
	return New(ErrCommandBlocked, reason).
		WithContext("command", command).
		WithSeverity(SeverityCritical).
		WithSolution("请使用安全的命令或联系管理员")
}

// ============================================================================
// 错误判断辅助函数
// ============================================================================

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Retryable
	}
	return false
}

// IsNotFound 判断是否是 NotFound 错误
func IsNotFound(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrNotFound
	}
	return false
}

// IsTimeout 判断是否是超时错误
func IsTimeout(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrTimeout
	}
	return false
}

// IsConnectionError 判断是否是连接错误
func IsConnectionError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Code {
		case ErrNATSConnection, ErrSSHConnection, ErrCloudProvider:
			return true
		}
	}
	return false
}

// GetCode 获取错误码
func GetCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrUnknown
}

// GetSeverity 获取严重程度
func GetSeverity(err error) Severity {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Severity
	}
	return SeverityError
}
