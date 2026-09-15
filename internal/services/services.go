package services

import (
	"database/sql"
	"log"
	"time"

	"foodrescue-api/internal/database"
)

type listingQuote struct {
	id             string
	initialPrice   float64
	minimumPrice   float64
	safeUntil      time.Time
	createdAt      time.Time
}

// ─────────────── DYNAMIC DISCOUNT ───────────────
// Harga turun bertahap dari initial_price ke minimum_price
// mendekati safe_until. Menjalankan update tiap 1 menit.
func StartDiscountScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		applyDynamicDiscount()
	}
}

func applyDynamicDiscount() {
	rows, err := database.DB.Query(
		`SELECT id, initial_price, minimum_price, safe_until, created_at
		 FROM food_listings WHERE status = 'active' AND current_price > minimum_price`)
	if err != nil {
		log.Printf("discount scheduler query error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var q listingQuote
		if err := rows.Scan(&q.id, &q.initialPrice, &q.minimumPrice, &q.safeUntil, &q.createdAt); err != nil {
			continue
		}

		newPrice := calculatePrice(q)
		if newPrice < q.minimumPrice {
			newPrice = q.minimumPrice
		}

		database.DB.Exec("UPDATE food_listings SET current_price = ? WHERE id = ?", newPrice, q.id)
	}
}

// formula: price = min + (initial - min) * (remaining_time / total_time)^2
func calculatePrice(q listingQuote) float64 {
	total := q.safeUntil.Sub(q.createdAt)
	if total <= 0 {
		return q.minimumPrice
	}

	remaining := time.Until(q.safeUntil)
	if remaining <= 0 {
		return q.minimumPrice
	}

	ratio := remaining.Seconds() / total.Seconds()
	if ratio > 1 {
		ratio = 1
	}

	discountFactor := ratio * ratio
	return q.minimumPrice + (q.initialPrice-q.minimumPrice)*discountFactor
}

// ─────────────── LISTING EXPIRY ───────────────
func StartExpiryScheduler() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		expireListings()
	}
}

func expireListings() {
	_, err := database.DB.Exec(
		`UPDATE food_listings SET status = 'expired'
		 WHERE status = 'active' AND safe_until < NOW()`)
	if err != nil {
		log.Printf("expiry scheduler error: %v", err)
	}
}

// ─────────────── IMPACT (kg saved) ───────────────
// SDK-free helper: hitung estimasi CO2 dari kg makanan terselamatkan
func EstimateCO2Saved(foodKg float64) float64 {
	return foodKg * 2.5 // kg CO2e per kg makanan (estimasi konservatif)
}

var _ = sql.ErrNoRows