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

// OrderView memperkaya Order dengan informasi listing & pengiriman yang
// dibutuhkan aplikasi mobile (nama barang, alamat toko, status trip).
type OrderView struct {
	models.Order
	Listing       *models.FoodListing `json:"listing"`
	PickupAddress string              `json:"pickup_address"`
	TripStatus    string              `json:"trip_status"`
	Courier       *CourierBrief       `json:"courier,omitempty"`
}

// CourierBrief — data kurir yang menangani pengiriman sebuah pesanan.
type CourierBrief struct {
	UserID             string  `json:"user_id"`
	Name               string  `json:"name"`
	VehicleType        string  `json:"vehicle_type"`
	VerificationStatus string  `json:"verification_status"`
	AverageRating      float64 `json:"average_rating"`
}

// TokoOrderView — ringkasan pesanan masuk untuk kasir toko.
type TokoOrderView struct {
	ID                string              `json:"id"`
	Code              string              `json:"code"`
	BuyerName         string              `json:"buyer_name"`
	ItemSummary       string              `json:"item_summary"`
	QuantityLabel     string              `json:"quantity_label"`
	TotalAmount       float64             `json:"total_amount"`
	PaymentStatus     string              `json:"payment_status"`
	FulfillmentMethod string              `json:"fulfillment_method"`
	OrderStatus       string              `json:"order_status"`
	PinPickup         string              `json:"pin_pickup"`
	EtaLabel          *string             `json:"eta_label"`
	DropoffPoint      *string             `json:"dropoff_point"`
	Listing           *models.FoodListing `json:"listing"`
}

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
			 VALUES (?, ?, 'mencari_kurir', ?, ?, ?, ?, ?, ?, ?, 8000, ?, ?)`,
			deliveryID, orderID, pickupLat, pickupLng, dropoffLat, dropoffLng,
			pickupCode, dropoffCode, time.Now().Add(5*time.Minute), time.Now(), time.Now(),
		)

		go findNearestCourier(deliveryID, listing.TokoID)
	}

	c.JSON(http.StatusCreated, gin.H{
		"order_id":          orderID,
		"confirmation_code": code,
		"total_amount":      totalAmount,
		"message":           "Order created successfully",
	})
}

// myTokoID mengambil id toko_profile milik user login saat ini.
func myTokoID(c *gin.Context) (string, bool) {
	var tokoID string
	if err := database.DB.QueryRow(
		"SELECT id FROM toko_profiles WHERE user_id = ?", c.GetString("user_id"),
	).Scan(&tokoID); err != nil {
		return "", false
	}
	return tokoID, true
}

// enrichOrderView melengkapi Order dengan listing (join toko) dan delivery.
func enrichOrderView(order models.Order) OrderView {
	var v OrderView
	v.Order = order

	var fl models.FoodListing
	err := database.DB.QueryRow(
		`SELECT fl.id, fl.toko_id, fl.name, fl.category, fl.description, fl.photo_url,
		        fl.initial_price, fl.minimum_price, fl.current_price, fl.stock_quantity,
		        fl.food_safety_notes, fl.safe_until, fl.pickup_start_time, fl.pickup_end_time,
		        fl.status, fl.created_at, fl.updated_at,
		        COALESCE(tp.address, '')
		 FROM food_listings fl
		 LEFT JOIN toko_profiles tp ON fl.toko_id = tp.id
		 WHERE fl.id = ?`, order.ListingID,
	).Scan(
		&fl.ID, &fl.TokoID, &fl.Name, &fl.Category, &fl.Description, &fl.PhotoURL,
		&fl.InitialPrice, &fl.MinimumPrice, &fl.CurrentPrice, &fl.StockQuantity,
		&fl.FoodSafetyNotes, &fl.SafeUntil, &fl.PickupStartTime, &fl.PickupEndTime,
		&fl.Status, &fl.CreatedAt, &fl.UpdatedAt, &v.PickupAddress,
	)
	if err == nil {
		v.Listing = &fl
	}

	var trip string
	if err := database.DB.QueryRow(
		"SELECT COALESCE(trip_status, '') FROM deliveries WHERE order_id = ?", order.ID,
	).Scan(&trip); err == nil {
		v.TripStatus = trip
	}

	var cb CourierBrief
	err = database.DB.QueryRow(
		`SELECT cp.user_id, COALESCE(u.full_name,''), COALESCE(cp.vehicle_type,''),
		        COALESCE(cp.verification_status,''), COALESCE(cp.average_rating,0)
		   FROM deliveries d
		   JOIN courier_profiles cp ON d.courier_id = cp.id
		   JOIN users u ON cp.user_id = u.id
		  WHERE d.order_id = ?`, order.ID,
	).Scan(&cb.UserID, &cb.Name, &cb.VehicleType, &cb.VerificationStatus, &cb.AverageRating)
	if err == nil {
		v.Courier = &cb
	}

	return v
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

	orders := make([]OrderView, 0)
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.ListingID, &o.UserID, &o.Quantity, &o.PriceAtPurchase, &o.TotalAmount,
			&o.FulfillmentMethod, &o.PaymentMethod, &o.PaymentStatus, &o.OrderStatus,
			&o.ConfirmationCode, &o.CreatedAt, &o.CompletedAt); err != nil {
			continue
		}
		orders = append(orders, enrichOrderView(o))
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

	c.JSON(http.StatusOK, gin.H{"order": enrichOrderView(o)})
}

// GetTokoOrders — pesanan masuk untuk kasir toko (role toko).
func GetTokoOrders(c *gin.Context) {
	tokoID, ok := myTokoID(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	rows, err := database.DB.Query(
		`SELECT o.id, o.listing_id, o.user_id, o.quantity, o.price_at_purchase, o.total_amount,
		        o.fulfillment_method, COALESCE(o.payment_method,''), o.payment_status, o.order_status,
		        o.confirmation_code, o.created_at, o.completed_at,
		        COALESCE(u.full_name,''), COALESCE(u.address_text,''),
		        fl.name, COALESCE(d.trip_status,''),
		        COALESCE(d.pickup_confirmation_code, '')
		 FROM orders o
		 JOIN food_listings fl ON o.listing_id = fl.id
		 JOIN users u ON o.user_id = u.id
		 LEFT JOIN deliveries d ON d.order_id = o.id
		 WHERE fl.toko_id = ?
		 ORDER BY o.created_at DESC`, tokoID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type row struct {
		order        models.Order
		buyerName    string
		buyerAddress string
		itemName     string
		tripStatus   string
		pinPickup    string
	}

	rws := make([]row, 0)
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.order.ID, &r.order.ListingID, &r.order.UserID, &r.order.Quantity, &r.order.PriceAtPurchase, &r.order.TotalAmount,
			&r.order.FulfillmentMethod, &r.order.PaymentMethod, &r.order.PaymentStatus, &r.order.OrderStatus,
			&r.order.ConfirmationCode, &r.order.CreatedAt, &r.order.CompletedAt,
			&r.buyerName, &r.buyerAddress, &r.itemName, &r.tripStatus, &r.pinPickup); err != nil {
			continue
		}
		rws = append(rws, r)
	}

	items := make([]TokoOrderView, 0, len(rws))
	for _, r := range rws {
		qtyLabel := fmt.Sprintf("%dx", r.order.Quantity)
		pin := r.pinPickup
		if pin == "" {
			pin = r.order.ConfirmationCode
		}
		tv := TokoOrderView{
			ID:                r.order.ID,
			Code:              r.order.ConfirmationCode,
			BuyerName:         r.buyerName,
			ItemSummary:       r.itemName,
			QuantityLabel:     qtyLabel,
			TotalAmount:       r.order.TotalAmount,
			PaymentStatus:     r.order.PaymentStatus,
			FulfillmentMethod: r.order.FulfillmentMethod,
			OrderStatus:       r.order.OrderStatus,
			PinPickup:         pin,
			Listing:           enrichListing(r.order.ListingID),
		}
		if r.buyerAddress != "" {
			dp := r.buyerAddress
			tv.DropoffPoint = &dp
		}
		if r.order.FulfillmentMethod == "diantar_kurir" && r.tripStatus != "" {
			eta := tripETA(r.tripStatus)
			tv.EtaLabel = &eta
		}
		items = append(items, tv)
	}

	if items == nil {
		items = []TokoOrderView{}
	}
	c.JSON(http.StatusOK, gin.H{"orders": items})
}

// VerifyPickup — kasir toko memverifikasi PIN serah terima dan melepas escrow.
func VerifyPickup(c *gin.Context) {
	orderID := c.Param("id")

	tokoID, ok := myTokoID(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	var req struct {
		PIN string `json:"pin" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Pastikan order milik listing toko ini.
	var orderStatus, expCode string
	var deliveryCode models.NullString
	row := database.DB.QueryRow(
		`SELECT o.order_status, o.confirmation_code, d.pickup_confirmation_code
		 FROM orders o
		 JOIN food_listings fl ON o.listing_id = fl.id
		 LEFT JOIN deliveries d ON d.order_id = o.id
		 WHERE o.id = ? AND fl.toko_id = ?`, orderID, tokoID,
	)
	if err := row.Scan(&orderStatus, &expCode, &deliveryCode); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	valid := req.PIN == expCode || (deliveryCode.Valid && req.PIN == deliveryCode.String)
	if !valid {
		c.JSON(http.StatusOK, gin.H{"valid": false, "message": "PIN tidak cocok"})
		return
	}

	if orderStatus == "dibatalkan" {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "message": "Pesanan sudah dibatalkan"})
		return
	}

	newStatus := "selesai"
	if orderStatus != "selesai" {
		database.DB.Exec(
			"UPDATE orders SET order_status = ?, payment_status = 'paid', completed_at = ? WHERE id = ?",
			newStatus, time.Now(), orderID,
		)
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Pesanan diverifikasi", "order_status": newStatus})
}

func enrichListing(listingID string) *models.FoodListing {
	var fl models.FoodListing
	err := database.DB.QueryRow(
		`SELECT id, toko_id, name, category, description, photo_url, initial_price, minimum_price,
		        current_price, stock_quantity, food_safety_notes, safe_until, pickup_start_time,
		        pickup_end_time, status, created_at, updated_at
		 FROM food_listings WHERE id = ?`, listingID,
	).Scan(
		&fl.ID, &fl.TokoID, &fl.Name, &fl.Category, &fl.Description, &fl.PhotoURL,
		&fl.InitialPrice, &fl.MinimumPrice, &fl.CurrentPrice, &fl.StockQuantity,
		&fl.FoodSafetyNotes, &fl.SafeUntil, &fl.PickupStartTime, &fl.PickupEndTime,
		&fl.Status, &fl.CreatedAt, &fl.UpdatedAt,
	)
	if err != nil {
		return nil
	}
	return &fl
}

func tripETA(tripStatus string) string {
	switch tripStatus {
	case "menuju_toko":
		return "Tiba di toko dalam 5 mnt"
	case "barang_diambil":
		return "Barang sudah diambil kurir"
	case "menuju_user":
		return "Sedang dalam perjalanan ke pembeli"
	case "diterima_user":
		return "Pesanan sampai & terverifikasi"
	default:
		return "Kurir sedang bertugas"
	}
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
