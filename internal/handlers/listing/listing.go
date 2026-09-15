package listing

import (
	"database/sql"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func CreateListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var tokoID string
	err := database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ? AND verification_status = 'approved'", userID).Scan(&tokoID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Toko profile not found or not verified"})
		return
	}

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

	var pickupStart, pickupEnd sql.NullTime
	if req.PickupStartTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05", req.PickupStartTime)
		pickupStart = sql.NullTime{Time: t, Valid: true}
	}
	if req.PickupEndTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05", req.PickupEndTime)
		pickupEnd = sql.NullTime{Time: t, Valid: true}
	}

	_, err = database.DB.Exec(
		`INSERT INTO food_listings (id, toko_id, name, category, description, photo_url, initial_price, minimum_price,
		 current_price, stock_quantity, food_safety_notes, safe_until, pickup_start_time, pickup_end_time, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		id, tokoID, req.Name, req.Category,
		sql.NullString{String: req.Description, Valid: req.Description != ""},
		sql.NullString{String: req.PhotoURL, Valid: req.PhotoURL != ""},
		req.InitialPrice, req.MinimumPrice, req.InitialPrice, req.StockQuantity,
		sql.NullString{String: req.FoodSafetyNotes, Valid: req.FoodSafetyNotes != ""},
		safeUntil, pickupStart, pickupEnd, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listing"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"listing_id": id, "message": "Listing created successfully"})
}

func GetListing(c *gin.Context) {
	id := c.Param("id")

	var listing models.FoodListing
	err := database.DB.QueryRow(
		`SELECT id, toko_id, name, category, description, photo_url, initial_price, minimum_price,
		        current_price, stock_quantity, food_safety_notes, safe_until, pickup_start_time,
		        pickup_end_time, status, created_at, updated_at
		 FROM food_listings WHERE id = ?`, id,
	).Scan(
		&listing.ID, &listing.TokoID, &listing.Name, &listing.Category, &listing.Description,
		&listing.PhotoURL, &listing.InitialPrice, &listing.MinimumPrice, &listing.CurrentPrice,
		&listing.StockQuantity, &listing.FoodSafetyNotes, &listing.SafeUntil,
		&listing.PickupStartTime, &listing.PickupEndTime, &listing.Status,
		&listing.CreatedAt, &listing.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"listing": listing})
}

func ListListings(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	category := c.Query("category")
	search := c.Query("search")

	offset := (page - 1) * limit

	query := `SELECT id, toko_id, name, category, description, photo_url, initial_price, minimum_price,
	          current_price, stock_quantity, safe_until, status, created_at
	          FROM food_listings WHERE status = 'active'`
	countQuery := "SELECT COUNT(*) FROM food_listings WHERE status = 'active'"

	var args []interface{}

	if category != "" {
		query += " AND category = ?"
		countQuery += " AND category = ?"
		args = append(args, category)
	}
	if search != "" {
		query += " AND name LIKE ?"
		countQuery += " AND name LIKE ?"
		args = append(args, "%"+search+"%")
	}

	var total int
	database.DB.QueryRow(countQuery, args...).Scan(&total)

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var listings []models.FoodListing
	for rows.Next() {
		var l models.FoodListing
		rows.Scan(&l.ID, &l.TokoID, &l.Name, &l.Category, &l.Description,
			&l.PhotoURL, &l.InitialPrice, &l.MinimumPrice, &l.CurrentPrice,
			&l.StockQuantity, &l.SafeUntil, &l.Status, &l.CreatedAt)
		listings = append(listings, l)
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, models.PaginatedResponse{
		Data:       listings,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

func UpdateListing(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var tokoID string
	err := database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	var listing models.FoodListing
	err = database.DB.QueryRow("SELECT id FROM food_listings WHERE id = ? AND toko_id = ?", id, tokoID).Scan(&listing.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found or not owned by you"})
		return
	}

	var req models.UpdateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		database.DB.Exec("UPDATE food_listings SET name = ? WHERE id = ?", req.Name, id)
	}
	if req.Category != "" {
		database.DB.Exec("UPDATE food_listings SET category = ? WHERE id = ?", req.Category, id)
	}
	if req.Description != "" {
		database.DB.Exec("UPDATE food_listings SET description = ? WHERE id = ?", req.Description, id)
	}
	if req.PhotoURL != "" {
		database.DB.Exec("UPDATE food_listings SET photo_url = ? WHERE id = ?", req.PhotoURL, id)
	}
	if req.InitialPrice > 0 {
		database.DB.Exec("UPDATE food_listings SET initial_price = ? WHERE id = ?", req.InitialPrice, id)
	}
	if req.MinimumPrice > 0 {
		database.DB.Exec("UPDATE food_listings SET minimum_price = ? WHERE id = ?", req.MinimumPrice, id)
	}
	if req.StockQuantity >= 0 {
		database.DB.Exec("UPDATE food_listings SET stock_quantity = ? WHERE id = ?", req.StockQuantity, id)
	}
	if req.Status != "" {
		database.DB.Exec("UPDATE food_listings SET status = ? WHERE id = ?", req.Status, id)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Listing updated successfully"})
}

func DeleteListing(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	var tokoID string
	database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoID)

	result, err := database.DB.Exec("DELETE FROM food_listings WHERE id = ? AND toko_id = ?", id, tokoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete listing"})
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found or not owned by you"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Listing deleted successfully"})
}

func GetMyListings(c *gin.Context) {
	userID := c.GetString("user_id")

	var tokoID string
	err := database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, toko_id, name, category, description, photo_url, initial_price, minimum_price,
		        current_price, stock_quantity, safe_until, status, created_at
		 FROM food_listings WHERE toko_id = ? ORDER BY created_at DESC`, tokoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var listings []models.FoodListing
	for rows.Next() {
		var l models.FoodListing
		rows.Scan(&l.ID, &l.TokoID, &l.Name, &l.Category, &l.Description,
			&l.PhotoURL, &l.InitialPrice, &l.MinimumPrice, &l.CurrentPrice,
			&l.StockQuantity, &l.SafeUntil, &l.Status, &l.CreatedAt)
		listings = append(listings, l)
	}

	c.JSON(http.StatusOK, gin.H{"listings": listings})
}
