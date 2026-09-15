package user

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func GetUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	err := database.DB.QueryRow(
		`SELECT id, email, full_name, photo_url, phone_number, auth_provider, role, is_ngo_verified,
		        trust_score, latitude, longitude, address_text, account_status, created_at, updated_at
		 FROM users WHERE id = ?`, userID,
	).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PhotoURL, &user.PhoneNumber,
		&user.AuthProvider, &user.Role, &user.IsNGOVerified, &user.TrustScore,
		&user.Latitude, &user.Longitude, &user.AddressText, &user.AccountStatus,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func GetMyImpact(c *gin.Context) {
	userID := c.GetString("user_id")

	var totalOrders int
	database.DB.QueryRow("SELECT COUNT(*) FROM orders WHERE user_id = ? AND order_status = 'selesai'", userID).Scan(&totalOrders)

	var totalFoodSavedKg float64
	database.DB.QueryRow("SELECT COALESCE(SUM(quantity * 0.5), 0) FROM orders WHERE user_id = ? AND order_status = 'selesai'", userID).Scan(&totalFoodSavedKg)

	var totalMoneySaved float64
	database.DB.QueryRow(`SELECT COALESCE(SUM(fl.initial_price * o.quantity - o.total_amount), 0)
		FROM orders o JOIN food_listings fl ON o.listing_id = fl.id
		WHERE o.user_id = ? AND o.order_status = 'selesai'`, userID).Scan(&totalMoneySaved)

	estimatedCO2 := totalFoodSavedKg * 2.5

	c.JSON(http.StatusOK, gin.H{
		"impact": gin.H{
			"total_orders":          totalOrders,
			"total_food_saved_kg":   totalFoodSavedKg,
			"total_money_saved":     totalMoneySaved,
			"estimated_co2_saved_kg": estimatedCO2,
		},
	})
}

func RegisterPaymentMethod(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Provider         string `json:"provider" binding:"required"`
		AccountReference string `json:"account_reference" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM payment_methods WHERE user_id = ?", userID).Scan(&count)
	isDefault := count == 0

	id := generateUUID()
	_, err := database.DB.Exec(
		`INSERT INTO payment_methods (id, user_id, provider, account_reference, is_default, created_at)
		 VALUES (?, ?, ?, ?, ?, NOW())`,
		id, userID, req.Provider, req.AccountReference, isDefault,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register payment method"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"payment_method_id": id, "message": "Payment method registered"})
}

func ListPaymentMethods(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := database.DB.Query(
		"SELECT id, user_id, provider, account_reference, is_default, created_at FROM payment_methods WHERE user_id = ?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var methods []models.PaymentMethod
	for rows.Next() {
		var m models.PaymentMethod
		rows.Scan(&m.ID, &m.UserID, &m.Provider, &m.AccountReference, &m.IsDefault, &m.CreatedAt)
		methods = append(methods, m)
	}

	c.JSON(http.StatusOK, gin.H{"payment_methods": methods})
}

func generateUUID() string {
	return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
}
