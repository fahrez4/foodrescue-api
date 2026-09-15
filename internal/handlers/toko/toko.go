package toko

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func GetMyTokoProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var profile models.TokoProfile
	err := database.DB.QueryRow(
		`SELECT id, user_id, business_name, business_category, address, latitude, longitude,
		        legal_document_url, operational_hours, verification_status, verified_by_admin_id,
		        verified_at, average_rating, created_at, updated_at
		 FROM toko_profiles WHERE user_id = ?`, userID,
	).Scan(
		&profile.ID, &profile.UserID, &profile.BusinessName, &profile.BusinessCategory,
		&profile.Address, &profile.Latitude, &profile.Longitude, &profile.LegalDocumentURL,
		&profile.OperationalHours, &profile.VerificationStatus, &profile.VerifiedByAdminID,
		&profile.VerifiedAt, &profile.AverageRating, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"toko_profile": profile})
}

func UpdateTokoProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		BusinessName     string `json:"business_name"`
		BusinessCategory string `json:"business_category"`
		Address          string `json:"address"`
		LegalDocumentURL string `json:"legal_document_url"`
		OperationalHours string `json:"operational_hours"`
		Latitude         float64 `json:"latitude"`
		Longitude        float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BusinessName != "" {
		database.DB.Exec("UPDATE toko_profiles SET business_name = ? WHERE user_id = ?", req.BusinessName, userID)
	}
	if req.BusinessCategory != "" {
		database.DB.Exec("UPDATE toko_profiles SET business_category = ? WHERE user_id = ?", req.BusinessCategory, userID)
	}
	if req.Address != "" {
		database.DB.Exec("UPDATE toko_profiles SET address = ?, latitude = ?, longitude = ? WHERE user_id = ?",
			req.Address, req.Latitude, req.Longitude, userID)
	}
	if req.LegalDocumentURL != "" {
		database.DB.Exec("UPDATE toko_profiles SET legal_document_url = ? WHERE user_id = ?", req.LegalDocumentURL, userID)
	}
	if req.OperationalHours != "" {
		database.DB.Exec("UPDATE toko_profiles SET operational_hours = ? WHERE user_id = ?", req.OperationalHours, userID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Toko profile updated successfully"})
}

func GetTokoByID(c *gin.Context) {
	tokoID := c.Param("id")

	var profile models.TokoProfile
	err := database.DB.QueryRow(
		`SELECT id, user_id, business_name, business_category, address, latitude, longitude,
		        legal_document_url, operational_hours, verification_status, verified_by_admin_id,
		        verified_at, average_rating, created_at, updated_at
		 FROM toko_profiles WHERE id = ?`, tokoID,
	).Scan(
		&profile.ID, &profile.UserID, &profile.BusinessName, &profile.BusinessCategory,
		&profile.Address, &profile.Latitude, &profile.Longitude, &profile.LegalDocumentURL,
		&profile.OperationalHours, &profile.VerificationStatus, &profile.VerifiedByAdminID,
		&profile.VerifiedAt, &profile.AverageRating, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"toko_profile": profile})
}

func ListApprovedTokos(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, user_id, business_name, business_category, address, latitude, longitude,
		        operational_hours, average_rating, created_at
		 FROM toko_profiles WHERE verification_status = 'approved'`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var tokos []models.TokoProfile
	for rows.Next() {
		var t models.TokoProfile
		rows.Scan(&t.ID, &t.UserID, &t.BusinessName, &t.BusinessCategory,
			&t.Address, &t.Latitude, &t.Longitude, &t.OperationalHours, &t.AverageRating, &t.CreatedAt)
		tokos = append(tokos, t)
	}

	c.JSON(http.StatusOK, gin.H{"tokos": tokos})
}

func GetSalesAnalytics(c *gin.Context) {
	userID := c.GetString("user_id")

	var tokoID string
	err := database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	var totalListings int
	database.DB.QueryRow("SELECT COUNT(*) FROM food_listings WHERE toko_id = ?", tokoID).Scan(&totalListings)

	var totalOrders int
	database.DB.QueryRow(`SELECT COUNT(*) FROM orders o
		JOIN food_listings fl ON o.listing_id = fl.id
		WHERE fl.toko_id = ? AND o.order_status = 'selesai'`, tokoID).Scan(&totalOrders)

	var totalRevenue float64
	database.DB.QueryRow(`SELECT COALESCE(SUM(o.total_amount), 0) FROM orders o
		JOIN food_listings fl ON o.listing_id = fl.id
		WHERE fl.toko_id = ? AND o.order_status = 'selesai' AND o.payment_status = 'paid'`, tokoID).Scan(&totalRevenue)

	var totalFoodSavedKg float64
	database.DB.QueryRow(`SELECT COALESCE(SUM(o.quantity * 0.5), 0) FROM orders o
		JOIN food_listings fl ON o.listing_id = fl.id
		WHERE fl.toko_id = ? AND o.order_status = 'selesai'`, tokoID).Scan(&totalFoodSavedKg)

	c.JSON(http.StatusOK, gin.H{
		"analytics": gin.H{
			"total_listings":      totalListings,
			"total_orders":        totalOrders,
			"total_revenue":       totalRevenue,
			"total_food_saved_kg": totalFoodSavedKg,
		},
	})
}

func GetTokoRatings(c *gin.Context) {
	userID := c.GetString("user_id")

	var tokoUserID sql.NullString
	err := database.DB.QueryRow("SELECT user_id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, order_id, rater_user_id, ratee_user_id, rating_target, score, COALESCE(review_text,''), created_at
		 FROM ratings WHERE ratee_user_id = ? AND rating_target = 'toko' ORDER BY created_at DESC`, userID)
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
