package asciinema

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// Asciinema 录制器
// ============================================================================

// Config 录制配置
type Config struct {
	OutputDir    string
	MinIOEndpoint string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket  string
}

// Recorder Asciinema 录制器
type Recorder struct {
	config      *Config
	mu          sync.Mutex
	recordings  map[string]*Recording
}

// Recording 录制会话
type Recording struct {
	ID          string
	DeploymentID string
	Command     string
	StartTime   time.Time
	EndTime     time.Time
	Status      string // recording, completed, failed
	CastFile    string // .cast 文件路径
	MinIOURL    string // MinIO 对象路径
	Duration    int64  // 毫秒
	mu          sync.Mutex
}

// NewRecorder 创建录制器
func NewRecorder(config *Config) *Recorder {
	if config == nil {
		config = &Config{
			OutputDir: "/tmp/asciinema",
		}
	}

	// 确保输出目录存在
	os.MkdirAll(config.OutputDir, 0755)

	return &Recorder{
		config:     config,
		recordings: make(map[string]*Recording),
	}
}

// StartRecording 开始录制
func (r *Recorder) StartRecording(deploymentID, command string) (*Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在
	if existing, ok := r.recordings[deploymentID]; ok {
		if existing.Status == "recording" {
			return existing, fmt.Errorf("recording already in progress")
		}
	}

	// 创建 cast 文件
	castFile := filepath.Join(r.config.OutputDir, fmt.Sprintf("%s-%d.cast", deploymentID, time.Now().Unix()))

	rec := &Recording{
		ID:           deploymentID,
		DeploymentID: deploymentID,
		Command:      command,
		StartTime:    time.Now(),
		Status:       "recording",
		CastFile:     castFile,
	}

	r.recordings[deploymentID] = rec

	// 启动 asciinema 录制
	go r.runAsciinema(rec)

	logrus.WithField("deployment_id", deploymentID).Info("Started asciinema recording")
	return rec, nil
}

// runAsciinema 运行 asciinema rec
func (r *Recorder) runAsciinema(rec *Recording) {
	defer func() {
		rec.mu.Lock()
		rec.EndTime = time.Now()
		rec.Duration = rec.EndTime.Sub(rec.StartTime).Milliseconds()
		rec.mu.Unlock()
	}()

	// 创建临时脚本文件来记录命令
	scriptFile := filepath.Join(r.config.OutputDir, fmt.Sprintf("%s-script.sh", rec.ID))
	if err := os.WriteFile(scriptFile, []byte(rec.Command), 0644); err != nil {
		rec.Status = "failed"
		logrus.Errorf("Failed to create script file: %v", err)
		return
	}

	// 执行 asciinema rec
	// 注意：asciinema rec 需要交互式终端
	// 实际实现可能需要使用 script 命令或其他方法
	cmd := exec.Command("asciinema", "rec", "--stdin", rec.CastFile)

	// 如果需要记录特定命令，可以使用 script 命令
	// cmd = exec.Command("script", "-f", "-c", rec.Command, rec.CastFile)

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	if err := cmd.Run(); err != nil {
		rec.Status = "failed"
		logrus.Errorf("Asciinema recording failed: %v", err)
		return
	}

	rec.Status = "completed"
	logrus.WithField("cast_file", rec.CastFile).Info("Asciinema recording completed")

	// 上传到 MinIO
	if err := r.uploadToMinIO(rec); err != nil {
		logrus.Errorf("Failed to upload recording to MinIO: %v", err)
	}
}

// StopRecording 停止录制
func (r *Recorder) StopRecording(deploymentID string) error {
	r.mu.Lock()
	rec, ok := r.recordings[deploymentID]
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("recording not found")
	}

	rec.mu.Lock()
	if rec.Status != "recording" {
		rec.mu.Unlock()
		return fmt.Errorf("recording not in progress")
	}
	rec.Status = "completed"
	rec.EndTime = time.Now()
	rec.Duration = rec.EndTime.Sub(rec.StartTime).Milliseconds()
	rec.mu.Unlock()

	logrus.WithField("deployment_id", deploymentID).Info("Stopped asciinema recording")
	return nil
}

// uploadToMinIO 上传到 MinIO
func (r *Recorder) uploadToMinIO(rec *Recording) error {
	if r.config.MinIOEndpoint == "" {
		// MinIO 未配置，跳过上传
		return nil
	}

	// 打开 cast 文件
	file, err := os.Open(rec.CastFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// 实际实现会使用 MinIO SDK 上传
	// 这里仅记录日志
	rec.MinIOURL = fmt.Sprintf("minio://%s/%s", r.config.MinIOBucket, filepath.Base(rec.CastFile))

	logrus.WithField("minio_url", rec.MinIOURL).Info("Recording uploaded to MinIO")
	return nil
}

// GetRecording 获取录制信息
func (r *Recorder) GetRecording(deploymentID string) (*Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, ok := r.recordings[deploymentID]
	if !ok {
		return nil, fmt.Errorf("recording not found")
	}

	return rec, nil
}

// GetCastContent 获取 cast 文件内容
func (r *Recorder) GetCastContent(deploymentID string) (string, error) {
	r.mu.Lock()
	rec, ok := r.recordings[deploymentID]
	r.mu.Unlock()

	if !ok {
		return "", fmt.Errorf("recording not found")
	}

	data, err := os.ReadFile(rec.CastFile)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// DeleteRecording 删除录制
func (r *Recorder) DeleteRecording(deploymentID string) error {
	r.mu.Lock()
	rec, ok := r.recordings[deploymentID]
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("recording not found")
	}

	// 删除 cast 文件
	os.Remove(rec.CastFile)

	r.mu.Lock()
	delete(r.recordings, deploymentID)
	r.mu.Unlock()

	logrus.WithField("deployment_id", deploymentID).Info("Deleted recording")
	return nil
}

// ListRecordings 列出所有录制
func (r *Recorder) ListRecordings() []*Recording {
	r.mu.Lock()
	defer r.mu.Unlock()

	recordings := make([]*Recording, 0, len(r.recordings))
	for _, rec := range r.recordings {
		recordings = append(recordings, rec)
	}
	return recordings
}

// ============================================================================
// Cast 文件解析
// ============================================================================

// CastHeader Cast 文件头
type CastHeader struct {
	Version      int                    `json:"version"`
	Width        int                    `json:"width"`
	Height       int                    `json:"height"`
	Timestamp    *int64                 `json:"timestamp,omitempty"`
	IdleTimeLimit *float64              `json:"idle_time_limit,omitempty"`
	Command      []string               `json:"command,omitempty"`
	Title        string                 `json:"title,omitempty"`
	Env          map[string]interface{} `json:"env,omitempty"`
	Theme        map[string]interface{} `json:"theme,omitempty"`
}

// CastEvent Cast 事件
type CastEvent struct {
	Time    float64 `json:"time"`    // 距离开始的时间（秒）
	Type    string  `json:"type"`    // 'o' = output, 'i' = input
	Data    string  `json:"data"`    // 输出/输入内容
}

// ParseCastFile 解析 Cast 文件
func ParseCastFile(path string) (*CastHeader, []CastEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var header *CastHeader
	var events []CastEvent

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// 第一行是 header
		if header == nil {
			if err := json.Unmarshal([]byte(line), header); err != nil {
				return nil, nil, fmt.Errorf("failed to parse header: %w", err)
			}
			continue
		}

		// 解析事件
		var event CastEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // 跳过无效行
		}
		events = append(events, event)
	}

	return header, events, scanner.Err()
}

// GetRecordingDuration 获取录制时长（毫秒）
func (r *Recording) GetDuration() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Duration
}

// GetStatus 获取录制状态
func (r *Recording) GetStatus() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Status
}

// GetCastFile 获取 cast 文件路径
func (r *Recording) GetCastFile() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.CastFile
}
