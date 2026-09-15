package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func GetDeliveryByOrder(c *gin.Context) {
	orderID := c.Param("order_id")

	var d models.Delivery
	err := database.DB.QueryRow(
		`SELECT id, order_id, courier_id, matching_status,
		        pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude,
		        trip_status, pickup_confirmation_code, dropoff_confirmation_code,
		        offer_expires_at, delivery_fee, created_at, updated_at
		 FROM deliveries WHERE order_id = ?`, orderID,
	).Scan(&d.ID, &d.OrderID, &d.CourierID, &d.MatchingStatus,
		&d.PickupLatitude, &d.PickupLongitude, &d.DropoffLatitude, &d.DropoffLongitude,
		&d.TripStatus, &d.PickupConfirmationCode, &d.DropoffConfirmationCode,
		&d.OfferExpiresAt, &d.DeliveryFee, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"delivery": d})
}

func GetMyDeliveries(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := database.DB.Query(
		`SELECT d.id, d.order_id, d.courier_id, d.matching_status,
		        d.pickup_latitude, d.pickup_longitude, d.dropoff_latitude, d.dropoff_longitude,
		        d.trip_status, d.delivery_fee, d.created_at
		 FROM deliveries d
		 JOIN courier_profiles cp ON d.courier_id = cp.id
		 WHERE cp.user_id = ? ORDER BY d.created_at DESC`, userID)
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
			&d.TripStatus, &d.DeliveryFee, &d.CreatedAt)
		deliveries = append(deliveries, d)
	}

	c.JSON(http.StatusOK, gin.H{"deliveries": deliveries})
}
