package pos

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

// IngestListing — terima item surplus dari mesin POS via API key toko.
// Endpoint: POST /api/v1/pos/listings (header X-API-Key).
func IngestListing(c *gin.Context) {
	tokoID := c.GetString("pos_toko_id")

	var req models.CreateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	safeUntil, err := time.Parse("2006-01-02T15:04:05", req.SafeUntil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid safe_until format. Use 2006-01-02T15:04:05"})
		return
	}

	id := uuid.New().String()
	now := time.Now()

	var pickupStart, pickupEnd models.NullTime
	if req.PickupStartTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05", req.PickupStartTime)
		pickupStart = models.NullTime{Time: t, Valid: true}
	}
	if req.PickupEndTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05", req.PickupEndTime)
		pickupEnd = models.NullTime{Time: t, Valid: true}
	}

	_, err = database.DB.Exec(
		`INSERT INTO food_listings (id, toko_id, name, category, description, photo_url, initial_price, minimum_price,
		 current_price, stock_quantity, food_safety_notes, safe_until, pickup_start_time, pickup_end_time, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		id, tokoID, req.Name, req.Category,
		models.NullString{String: req.Description, Valid: req.Description != ""},
		models.NullString{String: req.PhotoURL, Valid: req.PhotoURL != ""},
		req.InitialPrice, req.MinimumPrice, req.InitialPrice, req.StockQuantity,
		models.NullString{String: req.FoodSafetyNotes, Valid: req.FoodSafetyNotes != ""},
		safeUntil, pickupStart, pickupEnd, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listing"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"listing_id": id, "message": "Listing created via POS integration"})
}