package temporal

import (
	"fmt"
	"time"
)

// DeploymentWorkflowState 部署工作流状态
type DeploymentWorkflowState struct {
	WorkflowID    string                            `json:"workflow_id"`
	Input         DeploymentWorkflowInput           `json:"input"`
	CurrentState  string                            `json:"current_state"`
	StateHistory  []StateChange                     `json:"state_history"`
	StepStates    map[string]*StepState             `json:"step_states"`
	StartTime     time.Time                         `json:"start_time"`
	EndTime       time.Time                         `json:"end_time,omitempty"`
	CurrentStep   string                            `json:"current_step,omitempty"`
	Logs          []LogEntry                        `json:"logs"`
}

// StateChange 状态变更
type StateChange struct {
	FromState   string                 `json:"from_state,omitempty"`
	ToState     string                 `json:"to_state"`
	TriggeredBy string                 `json:"triggered_by"` // agent, user, system
	AgentName   string                 `json:"agent_name,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// StepState 步骤状态
type StepState struct {
	StepID      string                 `json:"step_id"`
	StepName    string                 `json:"step_name"`
	Status      string                 `json:"status"` // pending, running, success, failed, skipped
	StartTime   time.Time              `json:"start_time,omitempty"`
	EndTime     time.Time              `json:"end_time,omitempty"`
	Duration    time.Duration          `json:"duration_ms,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// LogEntry 日志条目
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Step      string                 `json:"step,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// NewDeploymentWorkflowState 创建新的工作流状态
func NewDeploymentWorkflowState() *DeploymentWorkflowState {
	return &DeploymentWorkflowState{
		CurrentState: StatePending,
		StateHistory: make([]StateChange, 0),
		StepStates:   make(map[string]*StepState),
		StartTime:    time.Now(),
		Logs:         make([]LogEntry, 0),
	}
}

// Transition 状态转换
func (s *DeploymentWorkflowState) Transition(newState string, triggeredBy string, agentName string) error {
	// 验证状态转换是否合法
	validTransitions := StateTransitions[s.CurrentState]
	isValid := false
	for _, valid := range validTransitions {
		if valid == newState {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("invalid state transition from %s to %s", s.CurrentState, newState)
	}

	// 记录状态变更
	change := StateChange{
		FromState:   s.CurrentState,
		ToState:     newState,
		TriggeredBy: triggeredBy,
		AgentName:   agentName,
		Timestamp:   time.Now(),
	}

	s.CurrentState = newState
	s.StateHistory = append(s.StateHistory, change)

	return nil
}

// StartStep 开始步骤
func (s *DeploymentWorkflowState) StartStep(stepID, stepName string) {
	s.CurrentStep = stepID
	s.StepStates[stepID] = &StepState{
		StepID:    stepID,
		StepName:  stepName,
		Status:    "running",
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	s.Log("info", fmt.Sprintf("Starting step: %s", stepName), stepID, nil)
}

// CompleteStep 完成步骤
func (s *DeploymentWorkflowState) CompleteStep(stepID string, output interface{}) {
	if step, ok := s.StepStates[stepID]; ok {
		step.Status = "success"
		step.EndTime = time.Now()
		step.Duration = step.EndTime.Sub(step.StartTime)
		step.Output = output

		s.Log("info", fmt.Sprintf("Step completed: %s", step.StepName), stepID, map[string]interface{}{
			"duration_ms": step.Duration.Milliseconds(),
		})
	}
	s.CurrentStep = ""
}

// FailStep 失败步骤
func (s *DeploymentWorkflowState) FailStep(stepID string, err error) {
	if step, ok := s.StepStates[stepID]; ok {
		step.Status = "failed"
		step.EndTime = time.Now()
		step.Duration = step.EndTime.Sub(step.StartTime)
		step.Error = err.Error()

		s.Log("error", fmt.Sprintf("Step failed: %s", step.StepName), stepID, map[string]interface{}{
			"error": err.Error(),
		})
	}
	s.CurrentStep = ""
}

// RecordStepSuccess 记录步骤成功
func (s *DeploymentWorkflowState) RecordStepSuccess(stepID string, output interface{}) {
	s.CompleteStep(stepID, output)
}

// RecordStepFailure 记录步骤失败
func (s *DeploymentWorkflowState) RecordStepFailure(stepID string, err error) {
	s.FailStep(stepID, err)
}

// RetryStep 重试步骤
func (s *DeploymentWorkflowState) RetryStep(stepID string) {
	if step, ok := s.StepStates[stepID]; ok {
		step.RetryCount++
		step.Status = "running"
		step.StartTime = time.Now()
		step.Error = ""

		s.Log("info", fmt.Sprintf("Retrying step: %s (attempt %d)", step.StepName, step.RetryCount), stepID, nil)
	}
}

// Log 添加日志
func (s *DeploymentWorkflowState) Log(level, message, step string, context map[string]interface{}) {
	s.Logs = append(s.Logs, LogEntry{
		Level:     level,
		Message:   message,
		Step:      step,
		Timestamp: time.Now(),
		Context:   context,
	})
}

// GetDuration 获取工作流持续时间
func (s *DeploymentWorkflowState) GetDuration() time.Duration {
	endTime := s.EndTime
	if endTime.IsZero() {
		endTime = time.Now()
	}
	return endTime.Sub(s.StartTime)
}

// GetStatus 获取当前状态摘要
func (s *DeploymentWorkflowState) GetStatus() map[string]interface{} {
	completed := 0
	failed := 0
	pending := 0

	for _, step := range s.StepStates {
		switch step.Status {
		case "success":
			completed++
		case "failed":
			failed++
		case "pending":
			pending++
		}
	}

	return map[string]interface{}{
		"workflow_id":     s.WorkflowID,
		"current_state":   s.CurrentState,
		"current_step":    s.CurrentStep,
		"total_steps":     len(s.StepStates),
		"completed_steps": completed,
		"failed_steps":    failed,
		"pending_steps":   pending,
		"duration_ms":     s.GetDuration().Milliseconds(),
		"start_time":      s.StartTime,
	}
}

// 状态定义
const (
	StatePending              = "PENDING"
	StateEnvPreparing         = "ENV_PREPARING"
	StateCodeFetching         = "CODE_FETCHING"
	StateDependencyInstalling = "DEPENDENCY_INSTALLING"
	StateBuilding             = "BUILDING"
	StateConfiguring          = "CONFIGURING"
	StateDeploying            = "DEPLOYING"
	StateVerifying            = "VERIFYING"
	StateCompleted            = "COMPLETED"
	StateRollingBack          = "ROLLING_BACK"
	StateFailed               = "FAILED"
	StatePaused               = "PAUSED"
)

// StateTransitions 状态转换定义
var StateTransitions = map[string][]string{
	StatePending:              {StateEnvPreparing, StateFailed},
	StateEnvPreparing:         {StateCodeFetching, StateRollingBack, StateFailed},
	StateCodeFetching:         {StateDependencyInstalling, StateRollingBack, StateFailed},
	StateDependencyInstalling: {StateBuilding, StateRollingBack, StateFailed, StatePaused},
	StateBuilding:             {StateConfiguring, StateRollingBack, StateFailed, StatePaused},
	StateConfiguring:          {StateDeploying, StateRollingBack, StateFailed},
	StateDeploying:            {StateVerifying, StateRollingBack, StateFailed},
	StateVerifying:            {StateCompleted, StateRollingBack, StateFailed},
	StateRollingBack:          {StateFailed, StatePending},
	StatePaused:               {StateBuilding, StateRollingBack, StateFailed},
}

// IsTerminalState 判断是否是终止状态
func IsTerminalState(state string) bool {
	return state == StateCompleted || state == StateFailed
}

// IsRetryableState 判断是否是可重试状态
func IsRetryableState(state string) bool {
	return state == StateFailed || state == StateRollingBack
}
