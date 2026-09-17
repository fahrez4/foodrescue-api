package kurir

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

// GetCourierPublicProfile mengembalikan user_id dari courier_profiles.id (publik, tanpa auth).
// Digunakan oleh user saat submit rating kurir: perlu user_id dari courier_profile_id.
func GetCourierPublicProfile(c *gin.Context) {
	courierProfileID := c.Param("courier_profile_id")
	if courierProfileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "courier_profile_id diperlukan"})
		return
	}

	var userID string
	err := database.DB.QueryRow(
		`SELECT user_id FROM courier_profiles WHERE id = ?`, courierProfileID,
	).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profil kurir tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"courier_profile_id": courierProfileID,
		"user_id":            userID,
	})
}

func GetMyCourierProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var profile models.CourierProfile
	err := database.DB.QueryRow(
		`SELECT id, user_id, vehicle_type, id_document_url, verification_status, verified_by_admin_id,
		        verified_at, is_online, current_latitude, current_longitude, last_location_update,
		        average_rating, total_earnings, created_at, updated_at
		 FROM courier_profiles WHERE user_id = ?`, userID,
	).Scan(
		&profile.ID, &profile.UserID, &profile.VehicleType, &profile.IDDocumentURL,
		&profile.VerificationStatus, &profile.VerifiedByAdminID, &profile.VerifiedAt,
		&profile.IsOnline, &profile.CurrentLatitude, &profile.CurrentLongitude,
		&profile.LastLocationUpdate, &profile.AverageRating, &profile.TotalEarnings,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Courier profile not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"courier_profile": profile})
}

func UpdateCourierProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		VehicleType  string `json:"vehicle_type"`
		IDDocumentURL string `json:"id_document_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.VehicleType != "" {
		database.DB.Exec("UPDATE courier_profiles SET vehicle_type = ? WHERE user_id = ?", req.VehicleType, userID)
	}
	if req.IDDocumentURL != "" {
		database.DB.Exec("UPDATE courier_profiles SET id_document_url = ? WHERE user_id = ?", req.IDDocumentURL, userID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Courier profile updated successfully"})
}

func ToggleOnline(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.ToggleOnlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.DB.Exec("UPDATE courier_profiles SET is_online = ? WHERE user_id = ?", req.IsOnline, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	status := "offline"
	if req.IsOnline {
		status = "online"
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

func UpdateLocation(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.UpdateCourierLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.DB.Exec(
		"UPDATE courier_profiles SET current_latitude = ?, current_longitude = ?, last_location_update = ? WHERE user_id = ?",
		req.Latitude, req.Longitude, time.Now(), userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Location updated successfully"})
}

func GetPendingDeliveries(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := database.DB.Query(
		`SELECT d.id, d.order_id, d.courier_id, d.matching_status,
		        d.pickup_latitude, d.pickup_longitude, d.dropoff_latitude, d.dropoff_longitude,
		        d.trip_status, d.pickup_confirmation_code, d.dropoff_confirmation_code,
		        d.offer_expires_at, d.delivery_fee, d.created_at, d.updated_at
		 FROM deliveries d
		 JOIN courier_profiles cp ON d.courier_id = cp.id
		 WHERE cp.user_id = ? AND d.matching_status = 'ditawarkan'
		 ORDER BY d.created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var deliveries []models.Delivery
	for rows.Next() {
		var d models.Delivery
		rows.Scan(&d.ID, &d.OrderID, &d.CourierID, &d.MatchingStatus,
			&d.PickupLatitude, &d.PickupLongitude, &d.DropoffLatitude, &d.DropoffLongitude,
			&d.TripStatus, &d.PickupConfirmationCode, &d.DropoffConfirmationCode,
			&d.OfferExpiresAt, &d.DeliveryFee, &d.CreatedAt, &d.UpdatedAt)
		deliveries = append(deliveries, d)
	}

	c.JSON(http.StatusOK, gin.H{"deliveries": deliveries})
}

func AcceptDelivery(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.AcceptDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var courierID string
	err := database.DB.QueryRow("SELECT id FROM courier_profiles WHERE user_id = ?", userID).Scan(&courierID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Courier profile not found"})
		return
	}

	_, err = database.DB.Exec(
		`UPDATE deliveries SET courier_id = ?, matching_status = 'diterima' WHERE id = ? AND matching_status = 'ditawarkan'`,
		courierID, req.DeliveryID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept delivery"})
		return
	}

	database.DB.Exec("UPDATE orders SET order_status = 'diantar' WHERE id = (SELECT order_id FROM deliveries WHERE id = ?)",
		req.DeliveryID)

	c.JSON(http.StatusOK, gin.H{"message": "Delivery accepted successfully"})
}

func UpdateTripStatus(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.UpdateTripStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Exec(
		`UPDATE deliveries d JOIN courier_profiles cp ON d.courier_id = cp.id
		 SET d.trip_status = ? WHERE cp.user_id = ? AND d.matching_status = 'diterima'`,
		req.TripStatus, userID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "Trip status updated"})
}

func ConfirmPickup(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.ConfirmPickupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pickupCode string
	err := database.DB.QueryRow(
		`SELECT d.pickup_confirmation_code FROM deliveries d
		 JOIN courier_profiles cp ON d.courier_id = cp.id
		 WHERE cp.user_id = ? AND d.matching_status = 'diterima'`, userID,
	).Scan(&pickupCode)
	if err != nil || pickupCode != req.ConfirmationCode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid confirmation code"})
		return
	}

	database.DB.Exec(
		`UPDATE deliveries d JOIN courier_profiles cp ON d.courier_id = cp.id
		 SET d.trip_status = 'barang_diambil' WHERE cp.user_id = ? AND d.matching_status = 'diterima'`, userID)

	c.JSON(http.StatusOK, gin.H{"message": "Pickup confirmed"})
}

func ConfirmDropoff(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.ConfirmDropoffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dropoffCode, orderID string
	err := database.DB.QueryRow(
		`SELECT d.dropoff_confirmation_code, d.order_id FROM deliveries d
		 JOIN courier_profiles cp ON d.courier_id = cp.id
		 WHERE cp.user_id = ? AND d.matching_status = 'diterima'`, userID,
	).Scan(&dropoffCode, &orderID)
	if err != nil || dropoffCode != req.ConfirmationCode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid confirmation code"})
		return
	}

	database.DB.Exec(`UPDATE deliveries d JOIN courier_profiles cp ON d.courier_id = cp.id
		SET d.trip_status = 'diterima_user' WHERE cp.user_id = ? AND d.matching_status = 'diterima'`, userID)
	database.DB.Exec("UPDATE orders SET order_status = 'selesai', completed_at = NOW() WHERE id = ?", orderID)

	c.JSON(http.StatusOK, gin.H{"message": "Dropoff confirmed, order completed"})
}

func GetCourierEarnings(c *gin.Context) {
	userID := c.GetString("user_id")

	var totalEarnings float64
	database.DB.QueryRow("SELECT total_earnings FROM courier_profiles WHERE user_id = ?", userID).Scan(&totalEarnings)

	rows, err := database.DB.Query(
		`SELECT d.id, d.order_id, d.delivery_fee, d.created_at
		 FROM deliveries d
		 JOIN courier_profiles cp ON d.courier_id = cp.id
		 WHERE cp.user_id = ? AND d.matching_status = 'diterima'
		 ORDER BY d.created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type EarningEntry struct {
		DeliveryID   string  `json:"delivery_id"`
		OrderID      string  `json:"order_id"`
		DeliveryFee  float64 `json:"delivery_fee"`
		CreatedAt    string  `json:"created_at"`
	}
	var earnings []EarningEntry
	for rows.Next() {
		var e EarningEntry
		rows.Scan(&e.DeliveryID, &e.OrderID, &e.DeliveryFee, &e.CreatedAt)
		earnings = append(earnings, e)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_earnings": totalEarnings,
		"history":        earnings,
	})
}
