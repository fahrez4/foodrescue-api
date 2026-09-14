package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"foodrescue-api/config"
	"foodrescue-api/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DonorSummary struct {
	ID       string `json:"id" example:"e7b93a0a-2fb1-4357-9d9b-d8927954b8d2"`
	Name     string `json:"name" example:"Restoran Berkah"`
	Phone    string `json:"phone" example:"081298765432"`
	PhotoURL string `json:"photo_url" example:"https://example.com/photos/donor.jpg"`
}

type FoodItemResponse struct {
	ID              string       `json:"id" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	DonorID         string       `json:"donor_id" example:"e7b93a0a-2fb1-4357-9d9b-d8927954b8d2"`
	Donor           DonorSummary `json:"donor"`
	Title           string       `json:"title" example:"Nasi Kotak Ayam Bakar"`
	Description     string       `json:"description" example:"Kelebihan porsi catering acara kantor, masih hangat dan higienis"`
	Category        string       `json:"category" example:"Makanan Berat"`
	PhotoURL        string       `json:"photo_url" example:"https://example.com/photos/food.jpg"`
	Quantity        string       `json:"quantity" example:"15 porsi"`
	Latitude        float64      `json:"latitude" example:"-6.2088"`
	Longitude       float64      `json:"longitude" example:"106.8456"`
	LocationAddress string       `json:"location_address" example:"Jl. Sudirman Kav. 50, Jakarta Pusat"`
	PickupStart     time.Time    `json:"pickup_start"`
	PickupEnd       time.Time    `json:"pickup_end"`
	Status          string       `json:"status" example:"available"`
	CreatedAt       time.Time    `json:"created_at"`
}

type CreateFoodRequest struct {
	Title           string  `json:"title" binding:"required" example:"Nasi Kotak Ayam Bakar"`
	Description     string  `json:"description" example:"Kelebihan porsi catering acara kantor, masih segar"`
	Category        string  `json:"category" binding:"required" example:"Makanan Berat"`
	Quantity        string  `json:"quantity" binding:"required" example:"15 porsi"`
	Latitude        float64 `json:"latitude" binding:"required" example:"-6.2088"`
	Longitude       float64 `json:"longitude" binding:"required" example:"106.8456"`
	LocationAddress string  `json:"location_address" example:"Jl. Sudirman Kav. 50, Jakarta Pusat"`
	PickupHours     int     `json:"pickup_hours" binding:"required,min=1" example:"4"`
}

type UpdateFoodRequest struct {
	Title           string  `json:"title" binding:"required" example:"Nasi Kotak Ayam Bakar Spesial"`
	Description     string  `json:"description" example:"Kelebihan porsi catering acara kantor, masih hangat dan lezat"`
	Category        string  `json:"category" binding:"required" example:"Makanan Berat"`
	Quantity        string  `json:"quantity" binding:"required" example:"20 porsi"`
	Latitude        float64 `json:"latitude" binding:"required" example:"-6.2088"`
	Longitude       float64 `json:"longitude" binding:"required" example:"106.8456"`
	LocationAddress string  `json:"location_address" example:"Jl. Sudirman Kav. 50, Jakarta Pusat"`
}

func GetFoods(c *gin.Context) {
	query := `
		SELECT 
			f.id, f.donor_id, u.name AS donor_name, COALESCE(u.phone, '') AS donor_phone, COALESCE(u.photo_url, '') AS donor_photo,
			f.title, COALESCE(f.description, '') AS description, f.category, COALESCE(f.photo_url, '') AS photo_url, f.quantity,
			f.latitude, f.longitude, COALESCE(f.location_address, '') AS location_address,
			f.pickup_start, f.pickup_end, f.status, f.created_at
		FROM food_items f
		JOIN users u ON f.donor_id = u.id
		WHERE f.status = 'available' AND f.pickup_end > NOW()
		ORDER BY f.created_at DESC
	`

	rows, err := config.DB.Query(query)
	if err != nil {
		log.Printf("[Error] GetFoods DB Query: %v", err)
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memuat daftar surplus makanan"))
		return
	}
	defer rows.Close()

	foods := make([]FoodItemResponse, 0)
	for rows.Next() {
		var item FoodItemResponse
		var donor DonorSummary

		err := rows.Scan(
			&item.ID,
			&item.DonorID,
			&donor.Name,
			&donor.Phone,
			&donor.PhotoURL,
			&item.Title,
			&item.Description,
			&item.Category,
			&item.PhotoURL,
			&item.Quantity,
			&item.Latitude,
			&item.Longitude,
			&item.LocationAddress,
			&item.PickupStart,
			&item.PickupEnd,
			&item.Status,
			&item.CreatedAt,
		)
	if err != nil {
		log.Printf("[Error] GetFoods Scan: %v", err)
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memproses data makanan"))
		return
	}

		donor.ID = item.DonorID
		item.Donor = donor
		foods = append(foods, item)
	}

	c.JSON(http.StatusOK, models.BuildResponse(true, "Berhasil memuat daftar surplus makanan", foods))
}

func GetFoodByID(c *gin.Context) {
	foodID := c.Param("id")

	query := `
		SELECT 
			f.id, f.donor_id, u.name AS donor_name, COALESCE(u.phone, '') AS donor_phone, COALESCE(u.photo_url, '') AS donor_photo,
			f.title, COALESCE(f.description, '') AS description, f.category, COALESCE(f.photo_url, '') AS photo_url, f.quantity,
			f.latitude, f.longitude, COALESCE(f.location_address, '') AS location_address,
			f.pickup_start, f.pickup_end, f.status, f.created_at
		FROM food_items f
		JOIN users u ON f.donor_id = u.id
		WHERE f.id = ?
	`

	var item FoodItemResponse
	var donor DonorSummary

	err := config.DB.QueryRow(query, foodID).Scan(
		&item.ID,
		&item.DonorID,
		&donor.Name,
		&donor.Phone,
		&donor.PhotoURL,
		&item.Title,
		&item.Description,
		&item.Category,
		&item.PhotoURL,
		&item.Quantity,
		&item.Latitude,
		&item.Longitude,
		&item.LocationAddress,
		&item.PickupStart,
		&item.PickupEnd,
		&item.Status,
		&item.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.BuildErrorResponse("Item makanan tidak ditemukan"))
			return
		}
		log.Printf("[Error] GetFoodByID DB: %v", err)
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memuat detail makanan"))
		return
	}

	donor.ID = item.DonorID
	item.Donor = donor

	c.JSON(http.StatusOK, models.BuildResponse(true, "Berhasil memuat detail makanan", item))
}

func GetMyFoods(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.BuildErrorResponse("User belum terautentikasi"))
		return
	}

	query := `
		SELECT 
			f.id, f.donor_id, u.name AS donor_name, COALESCE(u.phone, '') AS donor_phone, COALESCE(u.photo_url, '') AS donor_photo,
			f.title, COALESCE(f.description, '') AS description, f.category, COALESCE(f.photo_url, '') AS photo_url, f.quantity,
			f.latitude, f.longitude, COALESCE(f.location_address, '') AS location_address,
			f.pickup_start, f.pickup_end, f.status, f.created_at
		FROM food_items f
		JOIN users u ON f.donor_id = u.id
		WHERE f.donor_id = ?
		ORDER BY f.created_at DESC
	`

	rows, err := config.DB.Query(query, userID.(string))
	if err != nil {
		log.Printf("[Error] GetMyFoods DB Query: %v", err)
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memuat daftar donasi Anda"))
		return
	}
	defer rows.Close()

	foods := make([]FoodItemResponse, 0)
	for rows.Next() {
		var item FoodItemResponse
		var donor DonorSummary

		err := rows.Scan(
			&item.ID,
			&item.DonorID,
			&donor.Name,
			&donor.Phone,
			&donor.PhotoURL,
			&item.Title,
			&item.Description,
			&item.Category,
			&item.PhotoURL,
			&item.Quantity,
			&item.Latitude,
			&item.Longitude,
			&item.LocationAddress,
			&item.PickupStart,
			&item.PickupEnd,
			&item.Status,
			&item.CreatedAt,
		)
		if err != nil {
			log.Printf("[Error] GetMyFoods Scan: %v", err)
			c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memproses data donasi"))
			return
		}

		donor.ID = item.DonorID
		item.Donor = donor
		foods = append(foods, item)
	}

	c.JSON(http.StatusOK, models.BuildResponse(true, "Berhasil memuat daftar makanan donasi Anda", foods))
}

func CreateFood(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.BuildErrorResponse("User belum terautentikasi"))
		return
	}

	var req CreateFoodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("Data input tidak valid: "+err.Error()))
		return
	}

	itemID := uuid.New().String()

	query := `
		INSERT INTO food_items (
			id, donor_id, title, description, category, quantity,
			latitude, longitude, location_address,
			pickup_start, pickup_end, status, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?,
			NOW(), DATE_ADD(NOW(), INTERVAL ? HOUR), 'available', NOW()
		)
	`

	_, err := config.DB.Exec(
		query,
		itemID,
		userID.(string),
		req.Title,
		req.Description,
		req.Category,
		req.Quantity,
		req.Latitude,
		req.Longitude,
		req.LocationAddress,
		req.PickupHours,
	)

	if err != nil {
		log.Printf("[Error] CreateFood DB Exec: %v", err)
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal menyimpan data donasi makanan"))
		return
	}

	var pickupStart, pickupEnd, createdAt time.Time
	fetchQuery := "SELECT pickup_start, pickup_end, created_at FROM food_items WHERE id = ?"
	_ = config.DB.QueryRow(fetchQuery, itemID).Scan(&pickupStart, &pickupEnd, &createdAt)

	createdItem := gin.H{
		"id":               itemID,
		"donor_id":         userID,
		"title":            req.Title,
		"description":      req.Description,
		"category":         req.Category,
		"quantity":         req.Quantity,
		"latitude":         req.Latitude,
		"longitude":        req.Longitude,
		"location_address": req.LocationAddress,
		"pickup_start":     pickupStart,
		"pickup_end":       pickupEnd,
		"status":           "available",
		"created_at":       createdAt,
	}

	c.JSON(http.StatusCreated, models.BuildResponse(true, "Donasi makanan berhasil dibuat", createdItem))
}

func UpdateFood(c *gin.Context) {
	foodID := c.Param("id")
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.BuildErrorResponse("User belum terautentikasi"))
		return
	}
	userID := userIDVal.(string)

	var req UpdateFoodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("Data input tidak valid: "+err.Error()))
		return
	}

	var donorID, status string
	checkQuery := "SELECT donor_id, status FROM food_items WHERE id = ?"
	err := config.DB.QueryRow(checkQuery, foodID).Scan(&donorID, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.BuildErrorResponse("Item makanan tidak ditemukan"))
			return
		}
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memeriksa data makanan"))
		return
	}

	if donorID != userID {
		c.JSON(http.StatusForbidden, models.BuildErrorResponse("Anda hanya dapat mengubah makanan donasi milik Anda sendiri"))
		return
	}

	if status != "available" {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("Makanan yang sudah diklaim atau selesai tidak dapat diubah lagi"))
		return
	}

	updateQuery := `
		UPDATE food_items 
		SET title = ?, description = ?, category = ?, quantity = ?, latitude = ?, longitude = ?, location_address = ?
		WHERE id = ?
	`
	_, err = config.DB.Exec(updateQuery, req.Title, req.Description, req.Category, req.Quantity, req.Latitude, req.Longitude, req.LocationAddress, foodID)
	if err != nil {
		log.Printf("[Error] UpdateFood DB: %v", err)
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memperbarui data makanan"))
		return
	}

	c.JSON(http.StatusOK, models.BuildResponse(true, "Data donasi makanan berhasil diperbarui", gin.H{
		"id":               foodID,
		"title":            req.Title,
		"description":      req.Description,
		"category":         req.Category,
		"quantity":         req.Quantity,
		"latitude":         req.Latitude,
		"longitude":        req.Longitude,
		"location_address": req.LocationAddress,
	}))
}

func DeleteFood(c *gin.Context) {
	foodID := c.Param("id")
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.BuildErrorResponse("User belum terautentikasi"))
		return
	}
	userID := userIDVal.(string)

	var donorID, status string
	checkQuery := "SELECT donor_id, status FROM food_items WHERE id = ?"
	err := config.DB.QueryRow(checkQuery, foodID).Scan(&donorID, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.BuildErrorResponse("Item makanan tidak ditemukan"))
			return
		}
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memeriksa data makanan"))
		return
	}

	if donorID != userID {
		c.JSON(http.StatusForbidden, models.BuildErrorResponse("Anda hanya dapat menghapus donasi milik Anda sendiri"))
		return
	}

	if status != "available" {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("Makanan yang sudah diklaim atau selesai tidak dapat dihapus"))
		return
	}

	deleteQuery := "DELETE FROM food_items WHERE id = ?"
	_, err = config.DB.Exec(deleteQuery, foodID)
	if err != nil {
		log.Printf("[Error] DeleteFood DB: %v", err)
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal menghapus data makanan"))
		return
	}

	c.JSON(http.StatusOK, models.BuildResponse(true, "Donasi makanan berhasil dihapus", gin.H{"id": foodID}))
}
