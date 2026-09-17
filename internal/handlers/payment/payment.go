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
	midtransChargeBaseURL = "https://api.sandbox.midtrans.com/v4/charge"
	midtransIsProduction = false
)

type snapRequest struct {
	PaymentType        string             `json:"payment_type,omitempty"`
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

// CreateQRIS membuat Qris Charge ke Midtrans (payment_type=qris).
// Menghasilkan qr_string & URL gambar QR; bila server key belum diset,
// fallback ke mode "manual" agar alur UI tetap bisa diuji.
func CreateQRIS(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		OrderID string `json:"order_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order struct {
		ID            string
		TotalAmount   float64
		PaymentStatus string
	}
	err := database.DB.QueryRow(
		"SELECT id, total_amount, payment_status FROM orders WHERE id = ? AND user_id = ?",
		req.OrderID, userID,
	).Scan(&order.ID, &order.TotalAmount, &order.PaymentStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.PaymentStatus == "paid" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order already paid"})
		return
	}

	var user struct {
		FullName string
		Email    string
		Phone    string
	}
	database.DB.QueryRow(
		"SELECT full_name, email, COALESCE(phone_number,'') FROM users WHERE id = ?", userID,
	).Scan(&user.FullName, &user.Email, &user.Phone)

	serverKey := config.AppConfig.PaymentGatewayKey
	if serverKey == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":       "manual",
			"message":      "Payment gateway not configured. Use /orders/:id/confirm-payment to mark as paid (manual).",
			"order_id":     req.OrderID,
			"gross_amount": order.TotalAmount,
		})
		return
	}

	payload := snapRequest{
		TransactionDetails: transactionDetails{
			OrderID:     req.OrderID,
			GrossAmount: fmt.Sprintf("%.2f", order.TotalAmount),
		},
		ItemDetails: []itemDetails{{
			ID:       req.OrderID,
			Price:    order.TotalAmount,
			Quantity: 1,
			Name:     "Food Rescue Order",
		}},
		CustomerDetails: &customerDetails{
			FirstName: user.FullName,
			Email:     user.Email,
			Phone:     user.Phone,
		},
	}
	payload.PaymentType = "qris"

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", midtransChargeBaseURL, bytes.NewReader(body))
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

	var charge struct {
		StatusCode         string `json:"status_code"`
		TransactionID      string `json:"transaction_id"`
		QRString           string `json:"qr_string"`
		TransactionStatus  string `json:"transaction_status"`
		StatusMessage      string `json:"status_message"`
		AuthenticationMessage string `json:"authentication_message"`
		Actions            []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"actions"`
	}
	json.NewDecoder(resp.Body).Decode(&charge)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		msg := charge.StatusMessage
		if msg == "" && charge.AuthenticationMessage != "" {
			msg = charge.AuthenticationMessage
		}
		if msg == "" {
			msg = "Unknown payment gateway error"
		}
		c.JSON(resp.StatusCode, gin.H{"error": msg})
		return
	}

	qrImage := ""
	for _, a := range charge.Actions {
		if a.Name == "generate_qr_code" {
			qrImage = a.URL
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "pending",
		"transaction_id":   charge.TransactionID,
		"qr_string":        charge.QRString,
		"qr_image_url":     qrImage,
		"transaction_state": charge.TransactionStatus,
		"order_id":         req.OrderID,
		"gross_amount":     order.TotalAmount,
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