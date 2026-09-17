package rating

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func CreateRating(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.CreateRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Satu ulasan per pesanan per target (toko/kurir). Tolak duplikat.
	var existing int
	database.DB.QueryRow(
		"SELECT COUNT(*) FROM ratings WHERE rater_user_id = ? AND order_id = ? AND rating_target = ?",
		userID, req.OrderID, req.RatingTarget,
	).Scan(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Ulasan untuk pesanan ini sudah pernah dikirim",
			"code":    "already_rated",
			"message": "Ulasan hanya dapat dikirim satu kali.",
		})
		return
	}

	id := uuid.New().String()
	_, err := database.DB.Exec(
		`INSERT INTO ratings (id, order_id, rater_user_id, ratee_user_id, rating_target, score, review_text, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.OrderID, userID, req.RateeUserID, req.RatingTarget, req.Score,
		sqlNullString(req.ReviewText), time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rating"})
		return
	}

	updateAverageRating(req.RateeUserID, req.RatingTarget)

	c.JSON(http.StatusCreated, gin.H{"rating_id": id, "message": "Rating submitted"})
}

func GetRatingsForUser(c *gin.Context) {
	targetID := c.Param("id")
	target := c.Query("target")

	query := `SELECT id, order_id, rater_user_id, ratee_user_id, rating_target, score, COALESCE(review_text,''), created_at
	          FROM ratings WHERE ratee_user_id = ?`
	var args []interface{}
	args = append(args, targetID)

	if target != "" {
		query += " AND rating_target = ?"
		args = append(args, target)
	}
	query += " ORDER BY created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var ratings []models.Rating
	for rows.Next() {
		var r models.Rating
		rows.Scan(&r.ID, &r.OrderID, &r.RaterUserID, &r.RateeUserID, &r.RatingTarget, &r.Score, &r.ReviewText, &r.CreatedAt)
		ratings = append(ratings, r)
	}

	c.JSON(http.StatusOK, gin.H{"ratings": ratings})
}

func GetMyGivenRatings(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := database.DB.Query(
		`SELECT id, order_id, rater_user_id, ratee_user_id, rating_target, score, COALESCE(review_text,''), created_at
		 FROM ratings WHERE rater_user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var ratings []models.Rating
	for rows.Next() {
		var r models.Rating
		rows.Scan(&r.ID, &r.OrderID, &r.RaterUserID, &r.RateeUserID, &r.RatingTarget, &r.Score, &r.ReviewText, &r.CreatedAt)
		ratings = append(ratings, r)
	}

	c.JSON(http.StatusOK, gin.H{"ratings": ratings})
}

func updateAverageRating(userID, target string) {
	var avg float64
	database.DB.QueryRow("SELECT COALESCE(AVG(score), 0) FROM ratings WHERE ratee_user_id = ? AND rating_target = ?", userID, target).Scan(&avg)

	switch target {
	case "toko":
		database.DB.Exec("UPDATE toko_profiles SET average_rating = ? WHERE user_id = ?", avg, userID)
	case "kurir":
		database.DB.Exec("UPDATE courier_profiles SET average_rating = ? WHERE user_id = ?", avg, userID)
	}
}

func sqlNullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
