package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/sirupsen/logrus"
)

// Config Milvus 配置
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Host: "localhost",
		Port: 19530,
	}
}

// RAGKnowledgeBase RAG 知识库
type RAGKnowledgeBase struct {
	client      client.Client
	collection  string
	index       entity.Index
	searchParams *entity.SearchParam
}

// KnowledgeDocument 知识文档
type KnowledgeDocument struct {
	ID          string    `json:"id"`
	DeploymentID string   `json:"deployment_id"`
	CaseType    string    `json:"case_type"` // success, failure
	ErrorType   string    `json:"error_type,omitempty"`
	ErrorLog    string    `json:"error_log,omitempty"`
	RootCause   string    `json:"root_cause,omitempty"`
	Solution    string    `json:"solution,omitempty"`
	Commands    []string  `json:"commands,omitempty"`
	TechStack   string    `json:"tech_stack"`
	OS          string    `json:"os"`
	OSVersion   string    `json:"os_version"`
	Runtime     string    `json:"runtime"`
	QualityScore float64  `json:"quality_score"`
	Embedding   []float32 `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Doc      KnowledgeDocument
	Distance float32
}

// NewRAGKnowledgeBase 创建 RAG 知识库
func NewRAGKnowledgeBase(config *Config) (*RAGKnowledgeBase, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx := context.Background()
	c, err := client.NewGrpcClient(ctx, fmt.Sprintf("%s:%d", config.Host, config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %w", err)
	}

	kb := &RAGKnowledgeBase{
		client:     c,
		collection: "sera_knowledge_base",
	}

	// 初始化集合
	if err := kb.initCollection(ctx); err != nil {
		return nil, err
	}

	return kb, nil
}

// initCollection 初始化集合
func (kb *RAGKnowledgeBase) initCollection(ctx context.Context) error {
	// 检查集合是否存在
	has, err := kb.client.HasCollection(ctx, kb.collection)
	if err != nil {
		return err
	}

	if has {
		// 集合已存在，加载到内存
		return kb.client.LoadCollection(ctx, kb.collection, false)
	}

	// 创建集合
	schema := &entity.Schema{
		CollectionName: kb.collection,
		Description:    "Sera RAG Knowledge Base",
		AutoID:         true,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeVarChar,
				PrimaryKey: true,
				AutoID:     false,
				TypeParams: map[string]string{"max_length": "64"},
			},
			{
				Name:       "deployment_id",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "64"},
			},
			{
				Name:       "case_type",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "32"},
			},
			{
				Name:       "error_type",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "128"},
			},
			{
				Name:       "error_log",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "65535"},
			},
			{
				Name:       "root_cause",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "4096"},
			},
			{
				Name:       "solution",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "65535"},
			},
			{
				Name:       "commands",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "65535"},
			},
			{
				Name:       "tech_stack",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "256"},
			},
			{
				Name:       "os",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "64"},
			},
			{
				Name:       "os_version",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "64"},
			},
			{
				Name:       "runtime",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "64"},
			},
			{
				Name:       "quality_score",
				DataType:   entity.FieldTypeFloat,
			},
			{
				Name:       "created_at",
				DataType:   entity.FieldTypeInt64,
			},
			{
				Name:       "embedding",
				DataType:   entity.FieldTypeFloatVector,
				TypeParams: map[string]string{"dim": "768"}, // 使用 768 维向量（bge-small 等模型）
			},
		},
	}

	if err := kb.client.CreateCollection(ctx, schema, 1); err != nil {
		return err
	}

	// 创建索引
	idx, err := entity.NewIndexHNSW(entity.L2, 8, 200)
	if err != nil {
		return err
	}

	if err := kb.client.CreateIndex(ctx, kb.collection, "embedding", idx, false); err != nil {
		return err
	}

	// 加载集合到内存
	return kb.client.LoadCollection(ctx, kb.collection, false)
}

// Store 存储知识文档
func (kb *RAGKnowledgeBase) Store(ctx context.Context, doc *KnowledgeDocument) error {
	if doc.Embedding == nil {
		// 如果没有嵌入向量，使用简单文本作为 ID
		doc.Embedding = generateSimpleEmbedding(doc)
	}

	commandsJSON, _ := json.Marshal(doc.Commands)

	columns := []client.Column{
		client.NewVarCharColumn("id", []string{doc.ID}),
		client.NewVarCharColumn("deployment_id", []string{doc.DeploymentID}),
		client.NewVarCharColumn("case_type", []string{doc.CaseType}),
		client.NewVarCharColumn("error_type", []string{doc.ErrorType}),
		client.NewVarCharColumn("error_log", []string{doc.ErrorLog}),
		client.NewVarCharColumn("root_cause", []string{doc.RootCause}),
		client.NewVarCharColumn("solution", []string{doc.Solution}),
		client.NewVarCharColumn("commands", []string{string(commandsJSON)}),
		client.NewVarCharColumn("tech_stack", []string{doc.TechStack}),
		client.NewVarCharColumn("os", []string{doc.OS}),
		client.NewVarCharColumn("os_version", []string{doc.OSVersion}),
		client.NewVarCharColumn("runtime", []string{doc.Runtime}),
		client.NewFloatColumn("quality_score", []float32{float32(doc.QualityScore)}),
		client.NewInt64Column("created_at", []int64{doc.CreatedAt.Unix()}),
		client.NewFloatVectorColumn("embedding", 768, [][]float32{doc.Embedding}),
	}

	_, err := kb.client.Insert(ctx, kb.collection, "", columns...)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	logrus.WithField("id", doc.ID).Info("Knowledge document stored")
	return nil
}

// Search 搜索相似案例
func (kb *RAGKnowledgeBase) Search(ctx context.Context, query []float32, limit int) ([]SearchResult, error) {
	vector := entity.FloatVector(query)

	sp, err := entity.NewIndexHNSWSearchParam(30)
	if err != nil {
		return nil, err
	}

	results, err := kb.client.Search(ctx, kb.collection, nil, "", []string{"id", "deployment_id", "case_type", "error_type", "error_log", "root_cause", "solution", "commands", "tech_stack", "quality_score"}, []entity.Vector{vector}, "embedding", entity.L2, limit, sp)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	var searchResults []SearchResult
	for i := 0; i < results[0].ResultCount; i++ {
		id, _ := results[0].IDs.GetAsString(i)
		score := results[0].Scores[i]

		doc := KnowledgeDocument{
			ID: id,
		}

		// 解析其他字段（实际实现需要从结果中提取）
		searchResults = append(searchResults, SearchResult{
			Doc:      doc,
			Distance: score,
		})
	}

	return searchResults, nil
}

// SearchByText 通过文本搜索（需要先嵌入）
func (kb *RAGKnowledgeBase) SearchByText(ctx context.Context, text string, errorType string, limit int) ([]SearchResult, error) {
	// 生成查询嵌入
	queryEmbedding := generateSimpleEmbedding(&KnowledgeDocument{
		ErrorLog:  text,
		ErrorType: errorType,
	})

	return kb.Search(ctx, queryEmbedding, limit)
}

// Delete 删除知识文档
func (kb *RAGKnowledgeBase) Delete(ctx context.Context, id string) error {
	expr := fmt.Sprintf("id == '%s'", id)
	_, err := kb.client.Delete(ctx, kb.collection, "", expr)
	return err
}

// Count 获取文档数量
func (kb *RAGKnowledgeBase) Count(ctx context.Context) (int64, error) {
	stats, err := kb.client.GetCollectionStatistics(ctx, kb.collection)
	if err != nil {
		return 0, err
	}

	for _, rowCount := range stats.RowCount {
		return rowCount, nil
	}

	return 0, nil
}

// Close 关闭连接
func (kb *RAGKnowledgeBase) Close() error {
	return kb.client.Close()
}

// generateSimpleEmbedding 生成简单嵌入（占位符实现）
// 实际实现应该使用 BERT/BGE 等模型生成向量
func generateSimpleEmbedding(doc *KnowledgeDocument) []float32 {
	// 这是一个非常简化的实现，仅用于演示
	// 实际应该使用 HuggingFace 模型生成 768 维向量
	text := doc.ErrorLog + " " + doc.RootCause + " " + doc.Solution

	embedding := make([]float32, 768)
	for i, ch := range text {
		if i >= 768 {
			break
		}
		embedding[i] = float32(ch) / 256.0
	}

	return embedding
}

// ============================================================================
// 知识案例管理
// ============================================================================

// KnowledgeCaseManager 知识案例管理器
type KnowledgeCaseManager struct {
	kb *RAGKnowledgeBase
}

// NewKnowledgeCaseManager 创建知识案例管理器
func NewKnowledgeCaseManager(kb *RAGKnowledgeBase) *KnowledgeCaseManager {
	return &KnowledgeCaseManager{kb: kb}
}

// StoreSuccessCase 存储成功案例
func (km *KnowledgeCaseManager) StoreSuccessCase(ctx context.Context, deploymentID, solution string, commands []string, techStack string) error {
	doc := &KnowledgeDocument{
		ID:           fmt.Sprintf("success_%s", deploymentID),
		DeploymentID: deploymentID,
		CaseType:     "success",
		Solution:     solution,
		Commands:     commands,
		TechStack:    techStack,
		QualityScore: 1.0,
		CreatedAt:    time.Now(),
	}

	return km.kb.Store(ctx, doc)
}

// StoreFailureCase 存储失败案例
func (km *KnowledgeCaseManager) StoreFailureCase(ctx context.Context, deploymentID, errorType, errorLog, rootCause, solution string, commands []string) error {
	doc := &KnowledgeDocument{
		ID:           fmt.Sprintf("failure_%s", deploymentID),
		DeploymentID: deploymentID,
		CaseType:     "failure",
		ErrorType:    errorType,
		ErrorLog:     errorLog,
		RootCause:    rootCause,
		Solution:     solution,
		Commands:     commands,
		QualityScore: 0.8,
		CreatedAt:    time.Now(),
	}

	return km.kb.Store(ctx, doc)
}

// FindSimilarCases 查找相似案例
func (km *KnowledgeCaseManager) FindSimilarCases(ctx context.Context, errorLog, errorType string, limit int) ([]SearchResult, error) {
	return km.kb.SearchByText(ctx, errorLog, errorType, limit)
}
