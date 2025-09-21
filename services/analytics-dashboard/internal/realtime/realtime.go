package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Hub maintains the set of active connections and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	redis      *redis.Client
	mutex      sync.RWMutex
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   string
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	Topics   map[string]bool
	mutex    sync.RWMutex
}

// Message represents a real-time message
type Message struct {
	Type      MessageType `json:"type"`
	Topic     string      `json:"topic"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
	ClientID  string      `json:"client_id,omitempty"`
}

// MessageType represents different types of real-time messages
type MessageType string

const (
	MessageTypeData         MessageType = "data"
	MessageTypeAlert        MessageType = "alert"
	MessageTypeNotification MessageType = "notification"
	MessageTypeHeartbeat    MessageType = "heartbeat"
	MessageTypeSubscribe    MessageType = "subscribe"
	MessageTypeUnsubscribe  MessageType = "unsubscribe"
	MessageTypeError        MessageType = "error"
)

// SubscriptionRequest represents a subscription request
type SubscriptionRequest struct {
	Type   string   `json:"type"`
	Topics []string `json:"topics"`
}

// DataUpdate represents a data update message
type DataUpdate struct {
	WidgetID string      `json:"widget_id"`
	Data     interface{} `json:"data"`
	Source   string      `json:"source"`
}

// AlertMessage represents an alert message
type AlertMessage struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	Source      string    `json:"source"`
	Timestamp   time.Time `json:"timestamp"`
	AffectedSystems []string `json:"affected_systems"`
}

// NotificationMessage represents a notification message
type NotificationMessage struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	Priority  string    `json:"priority"`
	Timestamp time.Time `json:"timestamp"`
	Actions   []NotificationAction `json:"actions,omitempty"`
}

// NotificationAction represents an action in a notification
type NotificationAction struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// NewHub creates a new WebSocket hub
func NewHub(redis *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		redis:      redis,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("Client %s connected", client.ID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("Client %s disconnected", client.ID)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade connection"})
		return
	}

	userID := c.GetString("user_id") // Assume user ID is set by auth middleware
	if userID == "" {
		userID = "anonymous"
	}

	client := &Client{
		ID:     generateClientID(),
		UserID: userID,
		Hub:    h,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Topics: make(map[string]bool),
	}

	client.Hub.register <- client

	// Start goroutines for handling the connection
	go client.writePump()
	go client.readPump()
}

// BroadcastToTopic broadcasts a message to all clients subscribed to a topic
func (h *Hub) BroadcastToTopic(topic string, message *Message) error {
	message.Timestamp = time.Now()
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		client.mutex.RLock()
		subscribed := client.Topics[topic]
		client.mutex.RUnlock()

		if subscribed {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}

	// Also publish to Redis for other instances
	return h.publishToRedis(topic, data)
}

// BroadcastToUser broadcasts a message to a specific user
func (h *Hub) BroadcastToUser(userID string, message *Message) error {
	message.Timestamp = time.Now()
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}

	return nil
}

// SendDataUpdate sends a data update to subscribers
func (h *Hub) SendDataUpdate(widgetID string, data interface{}, source string) error {
	topic := fmt.Sprintf("widget:%s", widgetID)
	message := &Message{
		Type:  MessageTypeData,
		Topic: topic,
		Payload: DataUpdate{
			WidgetID: widgetID,
			Data:     data,
			Source:   source,
		},
	}

	return h.BroadcastToTopic(topic, message)
}

// SendAlert sends an alert to all subscribers
func (h *Hub) SendAlert(alert *AlertMessage) error {
	message := &Message{
		Type:    MessageTypeAlert,
		Topic:   "alerts",
		Payload: alert,
	}

	return h.BroadcastToTopic("alerts", message)
}

// SendNotification sends a notification to a user
func (h *Hub) SendNotification(userID string, notification *NotificationMessage) error {
	message := &Message{
		Type:    MessageTypeNotification,
		Topic:   "notifications",
		Payload: notification,
		UserID:  userID,
	}

	return h.BroadcastToUser(userID, message)
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetClientsByUser returns clients for a specific user
func (h *Hub) GetClientsByUser(userID string) []*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var clients []*Client
	for client := range h.clients {
		if client.UserID == userID {
			clients = append(clients, client)
		}
	}
	return clients
}

// publishToRedis publishes message to Redis for other instances
func (h *Hub) publishToRedis(topic string, data []byte) error {
	channel := fmt.Sprintf("realtime:%s", topic)
	return h.redis.Publish(context.Background(), channel, data).Err()
}

// SubscribeToRedis subscribes to Redis channels for distributed messaging
func (h *Hub) SubscribeToRedis() {
	pubsub := h.redis.PSubscribe(context.Background(), "realtime:*")
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		// Remove "realtime:" prefix to get the topic
		topic := msg.Channel[9:]
		
		// Broadcast to local clients
		h.broadcast <- []byte(msg.Payload)
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle subscription messages
		var subReq SubscriptionRequest
		if err := json.Unmarshal(message, &subReq); err == nil {
			c.handleSubscription(&subReq)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleSubscription handles subscription requests from clients
func (c *Client) handleSubscription(req *SubscriptionRequest) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch req.Type {
	case "subscribe":
		for _, topic := range req.Topics {
			c.Topics[topic] = true
			log.Printf("Client %s subscribed to topic %s", c.ID, topic)
		}
	case "unsubscribe":
		for _, topic := range req.Topics {
			delete(c.Topics, topic)
			log.Printf("Client %s unsubscribed from topic %s", c.ID, topic)
		}
	}

	// Send confirmation
	response := &Message{
		Type:     MessageTypeSubscribe,
		Topic:    "system",
		Payload:  fmt.Sprintf("Subscription updated: %s", req.Type),
		ClientID: c.ID,
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		close(c.Send)
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// Manager handles real-time operations
type Manager struct {
	hub *Hub
}

// NewManager creates a new real-time manager
func NewManager(redis *redis.Client) *Manager {
	hub := NewHub(redis)
	
	// Start the hub in a goroutine
	go hub.Run()
	
	// Start Redis subscription in a goroutine
	go hub.SubscribeToRedis()

	return &Manager{
		hub: hub,
	}
}

// GetHub returns the WebSocket hub
func (m *Manager) GetHub() *Hub {
	return m.hub
}

// HandleWebSocket handles WebSocket connections through the manager
func (m *Manager) HandleWebSocket(c *gin.Context) {
	m.hub.HandleWebSocket(c)
}