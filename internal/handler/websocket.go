package handler

import (
	"log"
	"net/http"
	"sync"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/shubhranka/spark_api/internal/data"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin is crucial for allowing connections from your Flutter app's domain/IP
	CheckOrigin: func(r *http.Request) bool {
		// For development, allow all connections.
		// For production, you MUST restrict this to your app's domain.
		return true
	},
}

// Hub manages all active client connections.
type Hub struct {
	// A map where the key is conversation_id and value is a map of client connections
	// The inner map's key is the client's connection pointer, value is bool (true)
	conversations map[string]map[*websocket.Conn]bool
	mu            sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		conversations: make(map[string]map[*websocket.Conn]bool),
	}
}

// Global hub instance
var WSHub = NewHub()

// HandleWebSocketConnection upgrades the HTTP request to a WebSocket connection.
func HandleWebSocketConnection(c *gin.Context) {
	// --- Authorization (Very Important!) ---
	// Real-time auth is slightly different. We'll pass the token as a query param.
	// Example URL: ws://localhost:8080/v1/ws/chat/CONVO_ID?token=FIREBASE_ID_TOKEN
	idToken := c.Query("token")
	if idToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "auth token is required"})
		return
	}

	// Get authClient and userModel from context
	authClient := c.MustGet("authClient").(*auth.Client)
	userModel := c.MustGet("userModel").(data.UserModel)
	convModel := c.MustGet("conversationModel").(data.ConversationModel)

	token, err := authClient.VerifyIDToken(c, idToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid auth token"})
		return
	}

	currentUser, err := userModel.GetByFirebaseUID(token.UID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "user not found"})
		return
	}

	conversationID := c.Param("id")
	if _, err := convModel.GetByID(conversationID, currentUser.ID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not a participant of this conversation"})
		return
	}

	// --- Upgrade to WebSocket ---
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Register the new client
	WSHub.addClient(conversationID, conn)
	defer WSHub.removeClient(conversationID, conn)

	// Listen for messages from this client
	for {
		// For this simple implementation, the client just listens.
		// Sending happens via the POST /messages endpoint, which then calls WSHub.Broadcast.
		// A more advanced implementation would read from the socket here.
		// This loop keeps the connection alive.
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Printf("Client disconnected: %v", err)
			break
		}
	}
}

func (h *Hub) addClient(conversationID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conversations[conversationID] == nil {
		h.conversations[conversationID] = make(map[*websocket.Conn]bool)
	}
	h.conversations[conversationID][conn] = true
	log.Printf("Client connected to conversation %s. Total clients: %d", conversationID, len(h.conversations[conversationID]))
}

func (h *Hub) removeClient(conversationID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, found := h.conversations[conversationID]; found {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(h.conversations, conversationID)
		}
	}
	log.Printf("Client removed from conversation %s.", conversationID)
}

// Broadcast sends a message to all clients in a specific conversation.
func (h *Hub) Broadcast(conversationID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, found := h.conversations[conversationID]; found {
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error broadcasting to client: %v", err)
			}
		}
	}
}
