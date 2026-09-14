package controllers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"

	"foodrescue-api/config"
	"foodrescue-api/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ClaimFood(c *gin.Context) {
	foodID := c.Param("id")
	if foodID == "" {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("ID makanan diperlukan"))
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.BuildErrorResponse("User belum terautentikasi"))
		return
	}
	userID := userIDVal.(string)

	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memulai transaksi database: "+err.Error()))
		return
	}
	defer tx.Rollback()

	var donorID, status string
	selectQuery := "SELECT donor_id, status FROM food_items WHERE id = ? FOR UPDATE"
	err = tx.QueryRow(selectQuery, foodID).Scan(&donorID, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.BuildErrorResponse("Item makanan tidak ditemukan"))
			return
		}
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal membaca status makanan: "+err.Error()))
		return
	}

	if donorID == userID {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("Anda tidak dapat mengklaim donasi makanan milik sendiri"))
		return
	}

	if status != "available" {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("Makanan sudah tidak tersedia atau telah diklaim sebelumnya"))
		return
	}

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal menghasilkan kode pengambilan"))
		return
	}
	pickupCode := fmt.Sprintf("%06d", n.Int64())

	claimID := uuid.New().String()

	updateFoodQuery := `
		UPDATE food_items 
		SET status = 'claimed', claimed_by = ?, pickup_code = ? 
		WHERE id = ?
	`
	_, err = tx.Exec(updateFoodQuery, userID, pickupCode, foodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memperbarui status makanan: "+err.Error()))
		return
	}

	insertClaimQuery := `
		INSERT INTO claims (id, item_id, donor_id, receiver_id, status, claimed_at) 
		VALUES (?, ?, ?, ?, 'pending', NOW())
	`
	_, err = tx.Exec(insertClaimQuery, claimID, foodID, donorID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal mencatat data klaim: "+err.Error()))
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal menyelesaikan transaksi klaim: "+err.Error()))
		return
	}

	claimResult := gin.H{
		"claim_id":    claimID,
		"food_id":     foodID,
		"pickup_code": pickupCode,
		"status":      "pending",
	}

	c.JSON(http.StatusOK, models.BuildResponse(true, "Makanan berhasil diklaim", claimResult))
}
