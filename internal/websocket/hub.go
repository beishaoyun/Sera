package websocket

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// WebSocket 管理器
// ============================================================================

// Upgrader WebSocket 升级器
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（生产环境应该限制）
	},
}

// Message WebSocket 消息
type Message struct {
	Type         string      `json:"type"` // deployment_progress, log, error, success
	DeploymentID string      `json:"deployment_id"`
	Step         string      `json:"step,omitempty"`
	State        string      `json:"state,omitempty"`
	Progress     int         `json:"progress,omitempty"` // 0-100
	Logs         string      `json:"logs,omitempty"`
	Error        string      `json:"error,omitempty"`
	Data         interface{} `json:"data,omitempty"`
	Timestamp    time.Time   `json:"timestamp"`
}

// Client WebSocket 客户端
type Client struct {
	ID           string
	Conn         *websocket.Conn
	Hub          *Hub
	DeploymentID string
	Send         chan *Message
	mu           sync.Mutex
}

// Hub 客户端中心
type Hub struct {
	clients    map[string]*Client
	deployment map[string]map[string]*Client // deployment_id -> client_id -> client
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		deployment: make(map[string]map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 运行 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if h.deployment[client.DeploymentID] == nil {
				h.deployment[client.DeploymentID] = make(map[string]*Client)
			}
			h.deployment[client.DeploymentID][client.ID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			if h.deployment[client.DeploymentID] != nil {
				delete(h.deployment[client.DeploymentID], client.ID)
				if len(h.deployment[client.DeploymentID]) == 0 {
					delete(h.deployment, client.DeploymentID)
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.deployment[message.DeploymentID]
			h.mu.RUnlock()

			for _, client := range clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
				}
			}
		}
	}
}

// WritePump 写入消息
func (c *Client) WritePump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.mu.Lock()
			if err := c.Conn.WriteJSON(message); err != nil {
				logrus.Errorf("Failed to write message: %v", err)
			}
			c.mu.Unlock()
		}
	}
}

// ReadPump 读取消息
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// ============================================================================
// Gin 处理器
// ============================================================================

// WSHandler WebSocket 处理器
type WSHandler struct {
	hub *Hub
}

// NewWSHandler 创建 WebSocket 处理器
func NewWSHandler() *WSHandler {
	hub := NewHub()
	go hub.Run()
	return &WSHandler{
		hub: hub,
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	deploymentID := c.Param("deployment_id")
	if deploymentID == "" {
		c.JSON(400, gin.H{"error": "deployment_id required"})
		return
	}

	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		ID:           uuid.New().String(),
		Conn:         conn,
		Hub:          h.hub,
		DeploymentID: deploymentID,
		Send:         make(chan *Message, 256),
	}

	h.hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

// BroadcastDeploymentProgress 广播部署进度
func (h *WSHandler) BroadcastDeploymentProgress(deploymentID, step, state string, progress int, logs string) {
	h.hub.broadcast <- &Message{
		Type:         "deployment_progress",
		DeploymentID: deploymentID,
		Step:         step,
		State:        state,
		Progress:     progress,
		Logs:         logs,
		Timestamp:    time.Now(),
	}
}

// BroadcastError 广播错误
func (h *WSHandler) BroadcastError(deploymentID, errorMsg string) {
	h.hub.broadcast <- &Message{
		Type:         "error",
		DeploymentID: deploymentID,
		Error:        errorMsg,
		Timestamp:    time.Now(),
	}
}

// BroadcastSuccess 广播成功
func (h *WSHandler) BroadcastSuccess(deploymentID string, data interface{}) {
	h.hub.broadcast <- &Message{
		Type:         "success",
		DeploymentID: deploymentID,
		Data:         data,
		Timestamp:    time.Now(),
	}
}

// GetHub 获取 Hub（用于外部访问）
func (h *WSHandler) GetHub() *Hub {
	return h.hub
}
