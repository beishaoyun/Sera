package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

// NATSClient NATS JetStream 客户端
type NATSClient struct {
	conn     *nats.Conn
	js       nats.JetStreamContext
	config   Config
	eventBus *EventBus
}

// Config NATS 配置
type Config struct {
	URL           string        `mapstructure:"url" json:"url"`                   // NATS 服务器地址
	MaxReconnect  int           `mapstructure:"max_reconnect" json:"max_reconnect"` // 最大重连次数
	ReconnectWait time.Duration `mapstructure:"reconnect_wait" json:"reconnect_wait"`
	Timeout       time.Duration `mapstructure:"timeout" json:"timeout"`
	ClusterID     string        `mapstructure:"cluster_id" json:"cluster_id"` // JetStream 集群 ID
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		URL:           "nats://localhost:4222",
		MaxReconnect:  10,
		ReconnectWait: 2 * time.Second,
		Timeout:       10 * time.Second,
		ClusterID:     "servermind",
	}
}

// Event 事件结构
type Event struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Subject     string                 `json:"subject"`
	Data        interface{}            `json:"data"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Correlation string                 `json:"correlation_id,omitempty"`
	Agent       string                 `json:"agent,omitempty"`
}

// NewEvent 创建新事件
func NewEvent(eventType, subject string, data interface{}, metadata map[string]string) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Subject:   subject,
		Data:      data,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
}

// NewNATSClient 创建 NATS 客户端
func NewNATSClient(ctx context.Context, config Config) (*NATSClient, error) {
	opts := []nats.Option{
		nats.Name("servermind"),
		nats.MaxReconnects(config.MaxReconnect),
		nats.ReconnectWait(config.ReconnectWait),
		nats.Timeout(config.Timeout),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logrus.WithError(err).Warn("NATS disconnected")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logrus.Info("NATS reconnected")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logrus.Warn("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// 创建 JetStream 上下文
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &NATSClient{
		conn:     nc,
		js:       js,
		config:   config,
		eventBus: NewEventBus(),
	}

	logrus.WithFields(logrus.Fields{
		"url": config.URL,
	}).Info("NATS client connected")

	return client, nil
}

// Publish 发布事件
func (c *NATSClient) Publish(ctx context.Context, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &nats.Msg{
		Subject: event.Subject,
		Data:    data,
		Header:  nats.Header{},
	}

	// 添加消息头
	msg.Header.Set("X-Event-ID", event.ID)
	msg.Header.Set("X-Event-Type", event.Type)
	msg.Header.Set("X-Timestamp", event.Timestamp.Format(time.RFC3339))

	if event.Correlation != "" {
		msg.Header.Set("X-Correlation-ID", event.Correlation)
	}

	if event.Agent != "" {
		msg.Header.Set("X-Agent", event.Agent)
	}

	// 发布到 JetStream
	_, err = c.js.PublishMsgContext(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"event_id":  event.ID,
		"event_type": event.Type,
		"subject":   event.Subject,
	}).Debug("Event published")

	return nil
}

// PublishRequest 发布请求/响应消息
func (c *NATSClient) PublishRequest(ctx context.Context, subject string, request interface{}, timeout time.Duration) (*nats.Msg, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := c.js.RequestContext(ctx, subject, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return msg, nil
}

// Subscribe 订阅主题
func (c *NATSClient) Subscribe(ctx context.Context, subject string, handler func(context.Context, *Event) error, opts ...SubscribeOption) (*Subscription, error) {
	options := DefaultSubscribeOptions()
	for _, opt := range opts {
		opt(options)
	}

	// 创建或获取 Stream
	streamName := options.StreamName
	if streamName == "" {
		streamName = "SERVERMIND_EVENTS"
	}

	// 尝试创建 Stream (如果不存在)
	_, err := c.js.StreamInfo(streamName)
	if err != nil {
		if err == nats.ErrStreamNotFound {
			// 创建 Stream
			_, err = c.js.AddStream(&nats.StreamConfig{
				Name:      streamName,
				Subjects:  []string{subject, subject + ".>"},
				Retention: nats.LimitsPolicy,
				Replicas:  1,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create stream: %w", err)
			}
			logrus.WithField("stream", streamName).Info("Stream created")
		} else {
			return nil, fmt.Errorf("failed to get stream info: %w", err)
		}
	}

	// 创建 Consumer
	consumerName := options.ConsumerName
	if consumerName == "" {
		consumerName = fmt.Sprintf("consumer_%s_%s", subject, uuid.New().String()[:8])
	}

	consumer, err := c.js.AddConsumer(streamName, &nats.ConsumerConfig{
		Durable:        consumerName,
		FilterSubject:  subject,
		DeliverPolicy:  nats.DeliverAllPolicy,
		AckPolicy:      nats.AckExplicitPolicy,
		MaxAckPending:  options.MaxAckPending,
		MaxDeliver:     options.MaxDeliver,
		AckWait:        options.AckWait,
		DeliverSubject: nats.NewInbox(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// 创建订阅
	sub, err := c.js.Subscribe("", func(msg *nats.Msg) {
		// 解析事件
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logrus.WithError(err).Error("Failed to unmarshal event")
			return
		}

		// 调用处理器
		if err := handler(ctx, &event); err != nil {
			logrus.WithError(err).Error("Event handler failed")
			// NAK 消息，稍后重试
			_ = msg.Nak()
			return
		}

		// ACK 消息
		_ = msg.Ack()
	}, nats.Bind(streamName, consumer.Name))

	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"subject":  subject,
		"stream":   streamName,
		"consumer": consumerName,
	}).Info("Subscription created")

	return &Subscription{
		sub:      sub,
		consumer: consumer,
		stream:   streamName,
	}, nil
}

// CreateDeadLetterQueue 创建死信队列
func (c *NATSClient) CreateDeadLetterQueue(ctx context.Context, streamName, subject string) error {
	// 创建 DLQ Stream
	_, err := c.js.AddStream(&nats.StreamConfig{
		Name:     streamName + "_DLQ",
		Subjects: []string{subject + ".dlq"},
		Retention: nats.LimitsPolicy,
		Replicas:  1,
		MaxMsgs:   100000, // 最多保留 10 万条消息
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return fmt.Errorf("failed to create DLQ stream: %w", err)
	}

	logrus.WithField("stream", streamName+"_DLQ").Info("DLQ stream created")
	return nil
}

// SendToDLQ 发送消息到死信队列
func (c *NATSClient) SendToDLQ(ctx context.Context, streamName string, originalMsg *nats.Msg, errorReason string) error {
	dlqSubject := streamName + "_DLQ"

	dlqEvent := map[string]interface{}{
		"original_subject": originalMsg.Subject,
		"error_reason":     errorReason,
		"original_data":    string(originalMsg.Data),
		"headers":          originalMsg.Header,
		"timestamp":        time.Now(),
	}

	data, err := json.Marshal(dlqEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ event: %w", err)
	}

	_, err = c.js.PublishContext(ctx, dlqSubject, data)
	if err != nil {
		return fmt.Errorf("failed to publish to DLQ: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"stream": streamName,
		"reason": errorReason,
	}).Warn("Message sent to DLQ")

	return nil
}

// RequestReply 请求/响应模式
func (c *NATSClient) RequestReply(ctx context.Context, subject string, request interface{}, timeout time.Duration) ([]byte, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	msg, err := c.js.RequestContext(ctx, subject, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return msg.Data, nil
}

// Close 关闭连接
func (c *NATSClient) Close() {
	if c.conn != nil {
		c.conn.Close()
		logrus.Info("NATS connection closed")
	}
}

// Subscription 订阅
type Subscription struct {
	sub      *nats.Subscription
	consumer *nats.ConsumerInfo
	stream   string
}

// Unsubscribe 取消订阅
func (s *Subscription) Unsubscribe() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}

// SubscribeOptions 订阅选项
type SubscribeOptions struct {
	StreamName    string
	ConsumerName  string
	MaxAckPending int
	MaxDeliver    int
	AckWait       time.Duration
	Durable       bool
}

type SubscribeOption func(*SubscribeOptions)

func DefaultSubscribeOptions() *SubscribeOptions {
	return &SubscribeOptions{
		MaxAckPending: 1000,
		MaxDeliver:    -1, // 无限重试
		AckWait:       30 * time.Second,
		Durable:       true,
	}
}

func WithStreamName(name string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.StreamName = name
	}
}

func WithConsumerName(name string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.ConsumerName = name
	}
}

func WithMaxAckPending(n int) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.MaxAckPending = n
	}
}

func WithMaxDeliver(n int) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.MaxDeliver = n
	}
}

// EventBus 事件总线
type EventBus struct {
	handlers map[string][]func(context.Context, *Event) error
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]func(context.Context, *Event) error),
	}
}

func (eb *EventBus) On(eventType string, handler func(context.Context, *Event) error) {
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Emit(ctx context.Context, event *Event) error {
	handlers, ok := eb.handlers[event.Type]
	if !ok {
		return nil
	}

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
