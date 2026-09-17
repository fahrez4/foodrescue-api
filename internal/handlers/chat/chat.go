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

	if !isOrderParticipant(userID, req.OrderID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a participant of this order"})
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

	c.JSON(http.StatusCreated, gin.H{"chat_id": id, "message": "Chat created"})
}

// CreateCommunityChat — chat antara toko dan Dapur Sosial/relawan untuk sebuah post komunitas.
func CreateCommunityChat(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.CreateCommunityChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !isCommunityPostParticipant(userID, req.CommunityPostID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya toko pemilik atau pengeklaim yang dapat membuka chat"})
		return
	}

	var exists int
	database.DB.QueryRow("SELECT COUNT(*) FROM chats WHERE community_post_id = ?", req.CommunityPostID).Scan(&exists)
	if exists > 0 {
		var chatID string
		database.DB.QueryRow("SELECT id FROM chats WHERE community_post_id = ?", req.CommunityPostID).Scan(&chatID)
		c.JSON(http.StatusOK, gin.H{"chat_id": chatID, "message": "Chat already exists"})
		return
	}

	id := uuid.New().String()
	_, err := database.DB.Exec(
		"INSERT INTO chats (id, community_post_id, created_at) VALUES (?, ?, ?)",
		id, req.CommunityPostID, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"chat_id": id, "message": "Chat created"})
}

func SendMessage(c *gin.Context) {
	userID := c.GetString("user_id")
	chatID := c.Param("id")

	if !canAccessChat(chatID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a participant of this chat"})
		return
	}

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
	userID := c.GetString("user_id")

	if !canAccessChat(chatID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a participant of this chat"})
		return
	}

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
		`SELECT c.id, c.order_id, c.community_post_id, c.created_at
		 FROM chats c
		 JOIN orders o ON c.order_id = o.id
		 WHERE o.user_id = ?
		 UNION
		 SELECT c.id, c.order_id, c.community_post_id, c.created_at
		 FROM chats c
		 JOIN orders o ON c.order_id = o.id
		 JOIN food_listings fl ON o.listing_id = fl.id
		 JOIN toko_profiles tp ON fl.toko_id = tp.id
		 WHERE tp.user_id = ?
		 UNION
		 SELECT c.id, c.order_id, c.community_post_id, c.created_at
		 FROM chats c
		 JOIN orders o ON c.order_id = o.id
		 JOIN deliveries d ON d.order_id = o.id
		 JOIN courier_profiles cp ON d.courier_id = cp.id
		 WHERE cp.user_id = ?
		 UNION
		 SELECT c.id, c.order_id, c.community_post_id, c.created_at
		 FROM chats c
		 JOIN community_posts cp ON c.community_post_id = cp.id
		 JOIN toko_profiles tp ON cp.toko_id = tp.id
		 WHERE tp.user_id = ?
		 UNION
		 SELECT c.id, c.order_id, c.community_post_id, c.created_at
		 FROM chats c
		 JOIN community_posts cp ON c.community_post_id = cp.id
		 WHERE cp.claimed_by_user_id = ?`,
		userID, userID, userID, userID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var ch models.Chat
		rows.Scan(&ch.ID, &ch.OrderID, &ch.CommunityPostID, &ch.CreatedAt)
		chats = append(chats, ch)
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

// ──────────────── HELPERS ────────────────

func isOrderParticipant(userID, orderID string) bool {
	var n int
	database.DB.QueryRow(`SELECT COUNT(*) FROM (
			SELECT o.user_id AS uid FROM orders o WHERE o.id = ?
			UNION
			SELECT tp.user_id FROM orders o JOIN food_listings fl ON o.listing_id = fl.id
			      JOIN toko_profiles tp ON fl.toko_id = tp.id WHERE o.id = ?
			UNION
			SELECT cp.user_id FROM deliveries d JOIN courier_profiles cp ON d.courier_id = cp.id WHERE d.order_id = ?
		) t WHERE t.uid = ?`, orderID, orderID, orderID, userID).Scan(&n)
	return n > 0
}

func isCommunityPostParticipant(userID, postID string) bool {
	var n int
	database.DB.QueryRow(`SELECT COUNT(*) FROM (
			SELECT tp.user_id AS uid FROM community_posts cp JOIN toko_profiles tp ON cp.toko_id = tp.id WHERE cp.id = ?
			UNION
			SELECT cp.claimed_by_user_id FROM community_posts cp WHERE cp.id = ? AND cp.claimed_by_user_id IS NOT NULL
		) t WHERE t.uid = ?`, postID, postID, userID).Scan(&n)
	return n > 0
}

func canAccessChat(chatID, userID string) bool {
	var orderID, communityPostID models.NullString
	err := database.DB.QueryRow("SELECT order_id, community_post_id FROM chats WHERE id = ?", chatID).Scan(&orderID, &communityPostID)
	if err != nil {
		return false
	}
	if orderID.Valid {
		return isOrderParticipant(userID, orderID.String)
	}
	if communityPostID.Valid {
		return isCommunityPostParticipant(userID, communityPostID.String)
	}
	return false
}