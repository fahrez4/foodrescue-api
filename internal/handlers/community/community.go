package community

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func CreateCommunityPost(c *gin.Context) {
	userID := c.GetString("user_id")

	var tokoID string
	err := database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	var req models.CreateCommunityPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	_, err = database.DB.Exec(
		`INSERT INTO community_posts (id, listing_id, toko_id, target_category, transport_fee, claim_status, created_at)
		 VALUES (?, ?, ?, ?, ?, 'tersedia', ?)`,
		id, req.ListingID, tokoID, req.TargetCategory, req.TransportFee, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create community post"})
		return
	}

	database.DB.Exec("UPDATE food_listings SET status = 'pushed_to_community' WHERE id = ?", req.ListingID)

	c.JSON(http.StatusCreated, gin.H{"community_post_id": id, "message": "Community post created"})
}

func ListCommunityPosts(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT cp.id, cp.listing_id, cp.toko_id, cp.target_category, cp.transport_fee,
		        cp.claim_status, cp.claimed_by_user_id, cp.claimed_at, cp.completed_at, cp.created_at,
		        fl.name, fl.photo_url, fl.description
		 FROM community_posts cp
		 JOIN food_listings fl ON cp.listing_id = fl.id
		 WHERE cp.claim_status = 'tersedia' ORDER BY cp.created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type CommunityPostWithListing struct {
		models.CommunityPost
		ListingName    string `json:"listing_name"`
		ListingPhoto   string `json:"listing_photo"`
		ListingDesc    string `json:"listing_description"`
	}

	var posts []CommunityPostWithListing
	for rows.Next() {
		var p CommunityPostWithListing
		rows.Scan(&p.ID, &p.ListingID, &p.TokoID, &p.TargetCategory, &p.TransportFee,
			&p.ClaimStatus, &p.ClaimedByUserID, &p.ClaimedAt, &p.CompletedAt, &p.CreatedAt,
			&p.ListingName, &p.ListingPhoto, &p.ListingDesc)
		posts = append(posts, p)
	}

	c.JSON(http.StatusOK, gin.H{"community_posts": posts})
}

func ClaimCommunityPost(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.ClaimCommunityPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := database.DB.Exec(
		`UPDATE community_posts SET claim_status = 'diklaim', claimed_by_user_id = ?, claimed_at = ?
		 WHERE id = ? AND claim_status = 'tersedia'`,
		userID, time.Now(), req.CommunityPostID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim"})
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Post not available for claiming"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Community post claimed successfully"})
}

func ConfirmCommunityPickup(c *gin.Context) {
	postID := c.Param("id")

	_, err := database.DB.Exec(
		`UPDATE community_posts SET claim_status = 'selesai', completed_at = ? WHERE id = ? AND claim_status = 'diklaim'`,
		time.Now(), postID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Community pickup confirmed"})
}
