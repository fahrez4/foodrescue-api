package order

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func CreateOrder(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var listing models.FoodListing
	err := database.DB.QueryRow(
		`SELECT id, toko_id, name, current_price, stock_quantity, status, safe_until
		 FROM food_listings WHERE id = ?`, req.ListingID,
	).Scan(&listing.ID, &listing.TokoID, &listing.Name, &listing.CurrentPrice,
		&listing.StockQuantity, &listing.Status, &listing.SafeUntil)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	if listing.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Listing is not active"})
		return
	}

	if listing.StockQuantity < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock"})
		return
	}

	if time.Now().After(listing.SafeUntil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Listing has expired"})
		return
	}

	orderID := uuid.New().String()
	code := generateConfirmationCode()
	priceAtPurchase := listing.CurrentPrice
	totalAmount := priceAtPurchase * float64(req.Quantity)

	_, err = database.DB.Exec(
		`INSERT INTO orders (id, listing_id, user_id, quantity, price_at_purchase, total_amount,
		 fulfillment_method, payment_method, payment_status, order_status, confirmation_code, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'unpaid', 'menunggu_pickup', ?, ?)`,
		orderID, req.ListingID, userID, req.Quantity, priceAtPurchase, totalAmount,
		req.FulfillmentMethod, models.NullString{String: req.PaymentMethod, Valid: req.PaymentMethod != ""},
		code, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	newStock := listing.StockQuantity - req.Quantity
	if newStock <= 0 {
		database.DB.Exec("UPDATE food_listings SET stock_quantity = 0, status = 'sold_out' WHERE id = ?", req.ListingID)
	} else {
		database.DB.Exec("UPDATE food_listings SET stock_quantity = ? WHERE id = ?", newStock, req.ListingID)
	}

	if req.FulfillmentMethod == "diantar_kurir" {
		deliveryID := uuid.New().String()
		var pickupLat, pickupLng, dropoffLat, dropoffLng models.NullFloat64
		var pickupCode, dropoffCode string
		pickupCode = generateConfirmationCode()
		dropoffCode = generateConfirmationCode()

		database.DB.QueryRow("SELECT latitude, longitude FROM toko_profiles WHERE id = ?", listing.TokoID).Scan(&pickupLat, &pickupLng)
		database.DB.QueryRow("SELECT latitude, longitude FROM users WHERE id = ?", userID).Scan(&dropoffLat, &dropoffLng)

		database.DB.Exec(
			`INSERT INTO deliveries (id, order_id, matching_status, pickup_latitude, pickup_longitude,
			 dropoff_latitude, dropoff_longitude, pickup_confirmation_code, dropoff_confirmation_code,
			 offer_expires_at, delivery_fee, created_at, updated_at)
			 VALUES (?, ?, 'mencari_kurir', ?, ?, ?, ?, ?, ?, ?, 5000, ?, ?)`,
			deliveryID, orderID, pickupLat, pickupLng, dropoffLat, dropoffLng,
			pickupCode, dropoffCode, time.Now().Add(5*time.Minute), time.Now(), time.Now(),
		)

		go findNearestCourier(deliveryID, listing.TokoID)
	}

	c.JSON(http.StatusCreated, gin.H{
		"order_id":         orderID,
		"confirmation_code": code,
		"total_amount":     totalAmount,
		"message":          "Order created successfully",
	})
}

func GetMyOrders(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := database.DB.Query(
		`SELECT id, listing_id, user_id, quantity, price_at_purchase, total_amount,
		        fulfillment_method, COALESCE(payment_method,''), payment_status, order_status,
		        confirmation_code, created_at, completed_at
		 FROM orders WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		rows.Scan(&o.ID, &o.ListingID, &o.UserID, &o.Quantity, &o.PriceAtPurchase, &o.TotalAmount,
			&o.FulfillmentMethod, &o.PaymentMethod, &o.PaymentStatus, &o.OrderStatus,
			&o.ConfirmationCode, &o.CreatedAt, &o.CompletedAt)
		orders = append(orders, o)
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	var o models.Order
	err := database.DB.QueryRow(
		`SELECT id, listing_id, user_id, quantity, price_at_purchase, total_amount,
		        fulfillment_method, COALESCE(payment_method,''), payment_status, order_status,
		        confirmation_code, created_at, completed_at
		 FROM orders WHERE id = ?`, orderID,
	).Scan(&o.ID, &o.ListingID, &o.UserID, &o.Quantity, &o.PriceAtPurchase, &o.TotalAmount,
		&o.FulfillmentMethod, &o.PaymentMethod, &o.PaymentStatus, &o.OrderStatus,
		&o.ConfirmationCode, &o.CreatedAt, &o.CompletedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": o})
}

func CancelOrder(c *gin.Context) {
	userID := c.GetString("user_id")
	orderID := c.Param("id")

	var o models.Order
	err := database.DB.QueryRow(
		"SELECT id, order_status, listing_id, quantity FROM orders WHERE id = ? AND user_id = ?",
		orderID, userID,
	).Scan(&o.ID, &o.OrderStatus, &o.ListingID, &o.Quantity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if o.OrderStatus != "menunggu_pickup" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel order in current status"})
		return
	}

	database.DB.Exec("UPDATE orders SET order_status = 'dibatalkan' WHERE id = ?", orderID)
	database.DB.Exec("UPDATE food_listings SET stock_quantity = stock_quantity + ?, status = 'active' WHERE id = ? AND status = 'sold_out'",
		o.Quantity, o.ListingID)

	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled"})
}

func ConfirmPayment(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		PaymentMethod string `json:"payment_method" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := database.DB.Exec(
		"UPDATE orders SET payment_status = 'paid', payment_method = ? WHERE id = ? AND user_id = ? AND payment_status = 'unpaid'",
		req.PaymentMethod, orderID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order not found or already paid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment confirmed"})
}

func FindNearestCourier(deliveryID, tokoID string) {
	findNearestCourier(deliveryID, tokoID)
}

func findNearestCourier(deliveryID, tokoID string) {
	var lat, lng models.NullFloat64
	database.DB.QueryRow("SELECT latitude, longitude FROM toko_profiles WHERE id = ?", tokoID).Scan(&lat, &lng)

	rows, err := database.DB.Query(
		`SELECT cp.id, cp.user_id, cp.current_latitude, cp.current_longitude
		 FROM courier_profiles cp
		 JOIN users u ON cp.user_id = u.id
		 WHERE cp.verification_status = 'approved' AND cp.is_online = TRUE
		 AND cp.current_latitude IS NOT NULL AND cp.current_longitude IS NOT NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	type Candidate struct {
		CourierProfileID string
		UserID           string
		Lat              float64
		Lng              float64
		Distance         float64
	}

	var candidates []Candidate
	for rows.Next() {
		var cand Candidate
		rows.Scan(&cand.CourierProfileID, &cand.UserID, &cand.Lat, &cand.Lng)
		if lat.Valid && lng.Valid {
			cand.Distance = haversine(lat.Float64, lng.Float64, cand.Lat, cand.Lng)
		}
		candidates = append(candidates, cand)
	}

	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].Distance < candidates[i].Distance {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	for _, cand := range candidates {
		offerID := uuid.New().String()
		_, err := database.DB.Exec(
			`INSERT INTO delivery_offers (id, delivery_id, courier_profile_id, offered_at, expires_at)
			 VALUES (?, ?, ?, NOW(), DATE_ADD(NOW(), INTERVAL 1 MINUTE))`,
			offerID, deliveryID, cand.CourierProfileID,
		)
		if err != nil {
			continue
		}

		database.DB.Exec(
			"UPDATE deliveries SET matching_status = 'ditawarkan' WHERE id = ?", deliveryID)

		time.Sleep(1 * time.Minute)

		var accepted int
		database.DB.QueryRow("SELECT COUNT(*) FROM delivery_offers WHERE delivery_id = ? AND accepted = TRUE", deliveryID).Scan(&accepted)
		if accepted > 0 {
			return
		}

		database.DB.Exec("UPDATE delivery_offers SET status = 'expired' WHERE id = ?", offerID)
	}

	database.DB.Exec("UPDATE deliveries SET matching_status = 'timeout' WHERE id = ?", deliveryID)
}

func generateConfirmationCode() string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := ""
	for i := 0; i < 6; i++ {
		code += string(chars[rand.Intn(len(chars))])
	}
	return code
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

var _ = fmt.Sprintf
