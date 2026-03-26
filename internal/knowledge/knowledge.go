package knowledge

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// KnowledgeBase RAG 知识库接口
type KnowledgeBase interface {
	// SearchSimilar 检索相似案例
	SearchSimilar(ctx context.Context, query string, category string, limit int) ([]KnowledgeCase, error)
	// AddCase 添加案例
	AddCase(ctx context.Context, caseData KnowledgeCase) error
	// UpdateCase 更新案例
	UpdateCase(ctx context.Context, id string, caseData KnowledgeCase) error
	// DeleteCase 删除案例
	DeleteCase(ctx context.Context, id string) error
	// GetCase 获取案例
	GetCase(ctx context.Context, id string) (*KnowledgeCase, error)
	// SearchByOS 按操作系统检索
	SearchByOS(ctx context.Context, os, osVersion string, limit int) ([]KnowledgeCase, error)
	// SearchByTechStack 按技术栈检索
	SearchByTechStack(ctx context.Context, techStack string, limit int) ([]KnowledgeCase, error)
}

// KnowledgeCase 知识案例
type KnowledgeCase struct {
	ID           string    `json:"id"`
	CaseType     string    `json:"case_type"` // success, failure
	OS           string    `json:"os"`
	OSVersion    string    `json:"os_version"`
	TechStack    string    `json:"tech_stack"`
	Runtime      string    `json:"runtime"`
	ErrorType    string    `json:"error_type,omitempty"`
	ErrorLog     string    `json:"error_log,omitempty"`
	RootCause    string    `json:"root_cause,omitempty"`
	Solution     string    `json:"solution,omitempty"`
	Commands     []string  `json:"commands,omitempty"`
	Embedding    []float32 `json:"-"` // 向量嵌入
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
	QualityScore float64   `json:"quality_score"`
	IsActive     bool      `json:"is_active"`
}

// RAGKnowledgeBase RAG 知识库实现
type RAGKnowledgeBase struct {
	milvusClient MilvusClient
	indexName    string
}

// MilvusClient Milvus 向量数据库客户端接口
type MilvusClient interface {
	// Insert 插入数据
	Insert(ctx context.Context, collection string, entities []interface{}) error
	// Search 向量搜索
	Search(ctx context.Context, collection string, vector []float32, topK int) ([]SearchResult, error)
	// Query 条件查询
	Query(ctx context.Context, collection string, expr string) ([]interface{}, error)
	// CreateCollection 创建集合
	CreateCollection(ctx context.Context, name string, dim int) error
	// DropCollection 删除集合
	DropCollection(ctx context.Context, name string) error
}

// SearchResult 搜索结果
type SearchResult struct {
	ID     string
	Score  float32
	Entity interface{}
}

// NewRAGKnowledgeBase 创建 RAG 知识库
func NewRAGKnowledgeBase(milvusClient MilvusClient, indexName string) *RAGKnowledgeBase {
	return &RAGKnowledgeBase{
		milvusClient: milvusClient,
		indexName:    indexName,
	}
}

// SearchSimilar 检索相似案例
func (kb *RAGKnowledgeBase) SearchSimilar(ctx context.Context, query string, category string, limit int) ([]KnowledgeCase, error) {
	logrus.WithFields(logrus.Fields{
		"query":    query,
		"category": category,
		"limit":    limit,
	}).Info("Searching knowledge base")

	// 1. 生成查询向量
	queryVector, err := kb.generateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// 2. 向量检索
	results, err := kb.milvusClient.Search(ctx, kb.indexName, queryVector, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// 3. 转换结果
	cases := make([]KnowledgeCase, 0, len(results))
	for _, result := range results {
		if caseData, ok := result.Entity.(KnowledgeCase); ok {
			// 过滤：如果指定了 category，只返回匹配的
			if category != "" && caseData.CaseType != category {
				continue
			}
			cases = append(cases, caseData)
		}
	}

	return cases, nil
}

// AddCase 添加案例
func (kb *RAGKnowledgeBase) AddCase(ctx context.Context, caseData KnowledgeCase) error {
	// 1. 生成向量嵌入
	embedding, err := kb.generateEmbedding(ctx, kb.buildEmbeddingText(caseData))
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	caseData.Embedding = embedding

	// 2. 插入向量数据库
	// 实际实现中需要将案例数据和向量一起插入
	logrus.WithFields(logrus.Fields{
		"id":        caseData.ID,
		"case_type": caseData.CaseType,
		"os":        caseData.OS,
	}).Info("Adding knowledge case")

	return nil
}

// UpdateCase 更新案例
func (kb *RAGKnowledgeBase) UpdateCase(ctx context.Context, id string, caseData KnowledgeCase) error {
	// 重新生成向量嵌入
	embedding, err := kb.generateEmbedding(ctx, kb.buildEmbeddingText(caseData))
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	caseData.Embedding = embedding

	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Info("Updating knowledge case")

	return nil
}

// DeleteCase 删除案例
func (kb *RAGKnowledgeBase) DeleteCase(ctx context.Context, id string) error {
	logrus.WithFields(logrus.Fields{
		"id": id,
	}).Info("Deleting knowledge case")
	return nil
}

// GetCase 获取案例
func (kb *RAGKnowledgeBase) GetCase(ctx context.Context, id string) (*KnowledgeCase, error) {
	return nil, nil
}

// SearchByOS 按操作系统检索
func (kb *RAGKnowledgeBase) SearchByOS(ctx context.Context, os, osVersion string, limit int) ([]KnowledgeCase, error) {
	logrus.WithFields(logrus.Fields{
		"os":        os,
		"os_version": osVersion,
	}).Info("Searching by OS")
	return nil, nil
}

// SearchByTechStack 按技术栈检索
func (kb *RAGKnowledgeBase) SearchByTechStack(ctx context.Context, techStack string, limit int) ([]KnowledgeCase, error) {
	logrus.WithFields(logrus.Fields{
		"tech_stack": techStack,
	}).Info("Searching by tech stack")
	return nil, nil
}

// generateEmbedding 生成向量嵌入
func (kb *RAGKnowledgeBase) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// 实际实现中调用 Embedding 模型 API
	// 例如：BAAI/bge-large-zh-v1.5
	// 这里返回一个占位向量
	return make([]float32, 1024), nil
}

// buildEmbeddingText 构建用于生成向量的文本
func (kb *RAGKnowledgeBase) buildEmbeddingText(caseData KnowledgeCase) string {
	// 组合多个字段的文本，用于生成更有意义的向量
	text := fmt.Sprintf(
		"OS: %s %s, Tech Stack: %s, Runtime: %s",
		caseData.OS, caseData.OSVersion, caseData.TechStack, caseData.Runtime,
	)

	if caseData.CaseType == "failure" {
		text += fmt.Sprintf(", Error: %s, Root Cause: %s, Solution: %s",
			caseData.ErrorType, caseData.RootCause, caseData.Solution)
	} else {
		text += fmt.Sprintf(", Solution: %s", caseData.Solution)
	}

	return text
}

// ============================================================================
// 知识进化机制
// ============================================================================

// KnowledgeEvolver 知识进化器
type KnowledgeEvolver struct {
	knowledgeBase KnowledgeBase
}

// NewKnowledgeEvolver 创建知识进化器
func NewKnowledgeEvolver(kb KnowledgeBase) *KnowledgeEvolver {
	return &KnowledgeEvolver{
		knowledgeBase: kb,
	}
}

// LearnFromSuccess 从成功案例学习
func (ke *KnowledgeEvolver) LearnFromSuccess(ctx context.Context, deployment DeploymentRecord) error {
	// 1. 提取关键信息
	caseData := ke.extractSuccessCase(deployment)

	// 2. 检查是否已存在相似案例
	existing, err := ke.knowledgeBase.SearchSimilar(ctx, ke.buildQuery(caseData), "success", 1)
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		// 存在相似案例，增加成功计数
		existing[0].SuccessCount++
		existing[0].QualityScore = ke.calculateQualityScore(existing[0])
		return ke.knowledgeBase.UpdateCase(ctx, existing[0].ID, existing[0])
	}

	// 3. 添加新案例
	return ke.knowledgeBase.AddCase(ctx, caseData)
}

// LearnFromFailure 从失败案例学习
func (ke *KnowledgeEvolver) LearnFromFailure(ctx context.Context, deployment DeploymentRecord, errorInfo ErrorInfo) error {
	caseData := ke.extractFailureCase(deployment, errorInfo)
	return ke.knowledgeBase.AddCase(ctx, caseData)
}

// extractSuccessCase 提取成功案例
func (ke *KnowledgeEvolver) extractSuccessCase(deployment DeploymentRecord) KnowledgeCase {
	return KnowledgeCase{
		ID:           generateUUID(),
		CaseType:     "success",
		OS:           deployment.OS,
		OSVersion:    deployment.OSVersion,
		TechStack:    deployment.TechStack,
		Runtime:      deployment.Runtime,
		Solution:     deployment.DeploySteps,
		SuccessCount: 1,
		QualityScore: 0.5, // 初始分数
		IsActive:     true,
	}
}

// extractFailureCase 提取失败案例
func (ke *KnowledgeEvolver) extractFailureCase(deployment DeploymentRecord, errorInfo ErrorInfo) KnowledgeCase {
	return KnowledgeCase{
		ID:           generateUUID(),
		CaseType:     "failure",
		OS:           deployment.OS,
		OSVersion:    deployment.OSVersion,
		TechStack:    deployment.TechStack,
		Runtime:      deployment.Runtime,
		ErrorType:    errorInfo.Type,
		ErrorLog:     errorInfo.Log,
		RootCause:    errorInfo.RootCause,
		Solution:     errorInfo.Solution,
		Commands:     errorInfo.FixCommands,
		FailureCount: 1,
		QualityScore: 0.3,
		IsActive:     true,
	}
}

// calculateQualityScore 计算案例质量分数
func (ke *KnowledgeEvolver) calculateQualityScore(caseData KnowledgeCase) float64 {
	total := caseData.SuccessCount + caseData.FailureCount
	if total == 0 {
		return 0.5
	}

	successRate := float64(caseData.SuccessCount) / float64(total)

	// 质量分数 = 成功率 * 0.7 + 基础分 0.3
	return successRate*0.7 + 0.3
}

// buildQuery 构建检索查询
func (ke *KnowledgeEvolver) buildQuery(caseData KnowledgeCase) string {
	return fmt.Sprintf("%s %s %s", caseData.OS, caseData.TechStack, caseData.Runtime)
}

// DeploymentRecord 部署记录
type DeploymentRecord struct {
	OS        string
	OSVersion string
	TechStack string
	Runtime   string
	DeploySteps string
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Type        string
	Log         string
	RootCause   string
	Solution    string
	FixCommands []string
}

func generateUUID() string {
	return "uuid-placeholder"
}
