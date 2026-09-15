package chat

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func CreateChat(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var exists int
	database.DB.QueryRow("SELECT COUNT(*) FROM chats WHERE order_id = ?", req.OrderID).Scan(&exists)
	if exists > 0 {
		var chatID string
		database.DB.QueryRow("SELECT id FROM chats WHERE order_id = ?", req.OrderID).Scan(&chatID)
		c.JSON(http.StatusOK, gin.H{"chat_id": chatID, "message": "Chat already exists"})
		return
	}

	id := uuid.New().String()
	_, err := database.DB.Exec(
		"INSERT INTO chats (id, order_id, created_at) VALUES (?, ?, ?)",
		id, req.OrderID, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	_ = userID

	c.JSON(http.StatusCreated, gin.H{"chat_id": id, "message": "Chat created"})
}

func SendMessage(c *gin.Context) {
	userID := c.GetString("user_id")
	chatID := c.Param("id")

	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	_, err := database.DB.Exec(
		"INSERT INTO messages (id, chat_id, sender_id, message_text, is_read, created_at) VALUES (?, ?, ?, ?, FALSE, ?)",
		id, chatID, userID, req.MessageText, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message_id": id, "message": "Message sent"})
}

func GetMessages(c *gin.Context) {
	chatID := c.Param("id")

	rows, err := database.DB.Query(
		`SELECT id, chat_id, sender_id, message_text, is_read, created_at
		 FROM messages WHERE chat_id = ? ORDER BY created_at ASC`, chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.MessageText, &m.IsRead, &m.CreatedAt)
		messages = append(messages, m)
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func GetMyChats(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := database.DB.Query(
		`SELECT c.id, c.order_id, c.created_at
		 FROM chats c
		 JOIN orders o ON c.order_id = o.id
		 WHERE o.user_id = ?
		 UNION
		 SELECT c.id, c.order_id, c.created_at
		 FROM chats c
		 JOIN orders o ON c.order_id = o.id
		 JOIN deliveries d ON d.order_id = o.id
		 JOIN courier_profiles cp ON d.courier_id = cp.id
		 WHERE cp.user_id = ?`, userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var ch models.Chat
		rows.Scan(&ch.ID, &ch.OrderID, &ch.CreatedAt)
		chats = append(chats, ch)
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}
