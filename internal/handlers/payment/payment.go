package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/config"
	"foodrescue-api/internal/database"
)

const (
	midtransSnapBaseURL = "https://app.sandbox.midtrans.com/snap/v1/transactions"
	midtransIsProduction = false
)

type snapRequest struct {
	TransactionDetails transactionDetails `json:"transaction_details"`
	ItemDetails        []itemDetails      `json:"item_details"`
	CustomerDetails    *customerDetails   `json:"customer_details,omitempty"`
}

type transactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount string `json:"gross_amount"`
}

type itemDetails struct {
	ID       string `json:"id"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Name     string  `json:"name"`
}

type customerDetails struct {
	FirstName string `json:"first_name,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

type snapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
	ErrorMessages []string `json:"error_messages"`
	StatusMessage string `json:"status_message"`
}

// CreateTransaction menghasilkan Snap token ke Midtrans sandbox.
// Jika server key belum diset, fallback ke status "unpaid" (manual/offline).
func CreateTransaction(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		OrderID  string `json:"order_id" binding:"required"`
		GrossAmt float64 `json:"gross_amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user struct {
		FullName     string
		Email        string
		Phone        string
	}
	err := database.DB.QueryRow(
		"SELECT full_name, email, COALESCE(phone_number,'') FROM users WHERE id = ?", userID,
	).Scan(&user.FullName, &user.Email, &user.Phone)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	serverKey := config.AppConfig.PaymentGatewayKey
	if serverKey == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":         "manual",
			"message":        "Payment gateway not configured. Use /orders/:id/confirm-payment to mark as paid (manual).",
			"order_id":       req.OrderID,
			"gross_amount":   req.GrossAmt,
		})
		return
	}

	payload := snapRequest{
		TransactionDetails: transactionDetails{
			OrderID:     req.OrderID,
			GrossAmount: fmt.Sprintf("%.2f", req.GrossAmt),
		},
		ItemDetails: []itemDetails{{
			ID:       req.OrderID,
			Price:    req.GrossAmt,
			Quantity: 1,
			Name:     "Food Rescue Order",
		}},
		CustomerDetails: &customerDetails{
			FirstName: user.FullName,
			Email:     user.Email,
			Phone:     user.Phone,
		},
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", midtransSnapBaseURL, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.SetBasicAuth(serverKey, "")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Payment gateway unreachable"})
		return
	}
	defer resp.Body.Close()

	var result snapResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		msg := result.StatusMessage
		if len(result.ErrorMessages) > 0 {
			msg = result.ErrorMessages[0]
		}
		if msg == "" {
			msg = "Unknown payment gateway error"
		}
		c.JSON(resp.StatusCode, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "pending",
		"snap_token":   result.Token,
		"redirect_url": result.RedirectURL,
		"order_id":     req.OrderID,
	})
}

// handleNotification memverifikasi signature dari webhook Midtrans,
// lalu meng-update status payment order.
func HandleNotification(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	orderID, _ := payload["order_id"].(string)
	statusCode, _ := payload["status_code"].(string)
	grossAmount, _ := payload["gross_amount"].(string)
	signatureKey, _ := payload["signature_key"].(string)

	serverKey := config.AppConfig.PaymentGatewayKey
	if serverKey != "" {
		mac := hmac.New(sha512.New, []byte(serverKey))
		mac.Write([]byte(orderID + statusCode + grossAmount + serverKey))
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(signatureKey)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
			return
		}
	}

	transactionStatus, _ := payload["transaction_status"].(string)

	newStatus := "unpaid"
	switch transactionStatus {
	case "capture", "settlement":
		if statusCode == "200" || statusCode == "201" {
			newStatus = "paid"
		}
	case "deny", "cancel", "expire", "failure":
		newStatus = "failed"
	case "pending", "challenge":
		newStatus = "unpaid"
	}

	if orderID != "" {
		database.DB.Exec(
			"UPDATE orders SET payment_status = ?, payment_method = 'midtrans' WHERE id = ?",
			newStatus, orderID,
		)
	}

	c.JSON(http.StatusOK, gin.H{"status": newStatus})
}

var _ = uuid.NewString