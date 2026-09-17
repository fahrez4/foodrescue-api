package toko

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func CreateTokoProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		BusinessName     string  `json:"business_name"`
		BusinessCategory string  `json:"business_category"`
		Address          string  `json:"address"`
		LegalDocumentURL string  `json:"legal_document_url"`
		OperationalHours string  `json:"operational_hours"`
		Latitude         float64 `json:"latitude"`
		Longitude        float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.BusinessName == "" || req.BusinessCategory == "" || req.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "business_name, business_category, and address are required"})
		return
	}

	var exists int
	err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM toko_profiles WHERE user_id = ?", userID,
	).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Toko profile already exists"})
		return
	}

	id := uuid.New().String()
	_, err = database.DB.Exec(
		`INSERT INTO toko_profiles (id, user_id, business_name, business_category, address,
		        latitude, longitude, legal_document_url, operational_hours, verification_status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		id, userID, req.BusinessName, req.BusinessCategory, req.Address,
		req.Latitude, req.Longitude, req.LegalDocumentURL, req.OperationalHours,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create toko profile"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Toko profile created, awaiting admin verification"})
}

func GetMyTokoProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var profile models.TokoProfile
	err := database.DB.QueryRow(
		`SELECT id, user_id, business_name, business_category, address, latitude, longitude,
		        legal_document_url, operational_hours, verification_status, verified_by_admin_id,
		        verified_at, average_rating, created_at, updated_at
		 FROM toko_profiles WHERE user_id = ?`, userID,
	).Scan(
		&profile.ID, &profile.UserID, &profile.BusinessName, &profile.BusinessCategory,
		&profile.Address, &profile.Latitude, &profile.Longitude, &profile.LegalDocumentURL,
		&profile.OperationalHours, &profile.VerificationStatus, &profile.VerifiedByAdminID,
		&profile.VerifiedAt, &profile.AverageRating, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"toko_profile": profile})
}

func UpdateTokoProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		BusinessName     string `json:"business_name"`
		BusinessCategory string `json:"business_category"`
		Address          string `json:"address"`
		LegalDocumentURL string `json:"legal_document_url"`
		OperationalHours string `json:"operational_hours"`
		Latitude         float64 `json:"latitude"`
		Longitude        float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BusinessName != "" {
		database.DB.Exec("UPDATE toko_profiles SET business_name = ? WHERE user_id = ?", req.BusinessName, userID)
	}
	if req.BusinessCategory != "" {
		database.DB.Exec("UPDATE toko_profiles SET business_category = ? WHERE user_id = ?", req.BusinessCategory, userID)
	}
	if req.Address != "" {
		database.DB.Exec("UPDATE toko_profiles SET address = ?, latitude = ?, longitude = ? WHERE user_id = ?",
			req.Address, req.Latitude, req.Longitude, userID)
	}
	if req.LegalDocumentURL != "" {
		database.DB.Exec("UPDATE toko_profiles SET legal_document_url = ? WHERE user_id = ?", req.LegalDocumentURL, userID)
	}
	if req.OperationalHours != "" {
		database.DB.Exec("UPDATE toko_profiles SET operational_hours = ? WHERE user_id = ?", req.OperationalHours, userID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Toko profile updated successfully"})
}

func GetTokoByID(c *gin.Context) {
	tokoID := c.Param("id")

	var profile models.TokoProfile
	err := database.DB.QueryRow(
		`SELECT id, user_id, business_name, business_category, address, latitude, longitude,
		        legal_document_url, operational_hours, verification_status, verified_by_admin_id,
		        verified_at, average_rating, created_at, updated_at
		 FROM toko_profiles WHERE id = ?`, tokoID,
	).Scan(
		&profile.ID, &profile.UserID, &profile.BusinessName, &profile.BusinessCategory,
		&profile.Address, &profile.Latitude, &profile.Longitude, &profile.LegalDocumentURL,
		&profile.OperationalHours, &profile.VerificationStatus, &profile.VerifiedByAdminID,
		&profile.VerifiedAt, &profile.AverageRating, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"toko_profile": profile})
}

func ListApprovedTokos(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, user_id, business_name, business_category, address, latitude, longitude,
		        operational_hours, verification_status, average_rating, created_at
		 FROM toko_profiles WHERE verification_status = 'approved'`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var tokos []models.TokoProfile
	for rows.Next() {
		var t models.TokoProfile
		rows.Scan(&t.ID, &t.UserID, &t.BusinessName, &t.BusinessCategory,
			&t.Address, &t.Latitude, &t.Longitude, &t.OperationalHours, &t.VerificationStatus, &t.AverageRating, &t.CreatedAt)
		tokos = append(tokos, t)
	}

	c.JSON(http.StatusOK, gin.H{"tokos": tokos})
}

func GetSalesAnalytics(c *gin.Context) {
	userID := c.GetString("user_id")

	var tokoID string
	err := database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	var totalListings int
	database.DB.QueryRow("SELECT COUNT(*) FROM food_listings WHERE toko_id = ?", tokoID).Scan(&totalListings)

	var totalOrders int
	database.DB.QueryRow(`SELECT COUNT(*) FROM orders o
		JOIN food_listings fl ON o.listing_id = fl.id
		WHERE fl.toko_id = ? AND o.order_status = 'selesai'`, tokoID).Scan(&totalOrders)

	var totalRevenue float64
	database.DB.QueryRow(`SELECT COALESCE(SUM(o.total_amount), 0) FROM orders o
		JOIN food_listings fl ON o.listing_id = fl.id
		WHERE fl.toko_id = ? AND o.order_status = 'selesai' AND o.payment_status = 'paid'`, tokoID).Scan(&totalRevenue)

	var totalFoodSavedKg float64
	database.DB.QueryRow(`SELECT COALESCE(SUM(o.quantity * 0.5), 0) FROM orders o
		JOIN food_listings fl ON o.listing_id = fl.id
		WHERE fl.toko_id = ? AND o.order_status = 'selesai'`, tokoID).Scan(&totalFoodSavedKg)

	c.JSON(http.StatusOK, gin.H{
		"analytics": gin.H{
			"total_listings":      totalListings,
			"total_orders":        totalOrders,
			"total_revenue":       totalRevenue,
			"total_food_saved_kg": totalFoodSavedKg,
		},
	})
}

func GetTokoRatings(c *gin.Context) {
	userID := c.GetString("user_id")

	var tokoUserID models.NullString
	err := database.DB.QueryRow("SELECT user_id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, order_id, rater_user_id, ratee_user_id, rating_target, score, COALESCE(review_text,''), created_at
		 FROM ratings WHERE ratee_user_id = ? AND rating_target = 'toko' ORDER BY created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var ratings []models.Rating
	for rows.Next() {
		var r models.Rating
		rows.Scan(&r.ID, &r.OrderID, &r.RaterUserID, &r.RateeUserID, &r.RatingTarget, &r.Score, &r.ReviewText, &r.CreatedAt)
		ratings = append(ratings, r)
	}

	c.JSON(http.StatusOK, gin.H{"ratings": ratings})
}

// ──────────────── REKENING PENGIRIM DANA ────────────────

func GetMyBankAccounts(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, toko_id, bank_name, bank_code, account_number, account_holder_name, is_primary, created_at, updated_at
		 FROM toko_bank_accounts WHERE toko_id = ? ORDER BY is_primary DESC, created_at ASC`, tokoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var accounts []models.TokoBankAccount
	for rows.Next() {
		var a models.TokoBankAccount
		rows.Scan(&a.ID, &a.TokoID, &a.BankName, &a.BankCode, &a.AccountNumber,
			&a.AccountHolderName, &a.IsPrimary, &a.CreatedAt, &a.UpdatedAt)
		accounts = append(accounts, a)
	}

	c.JSON(http.StatusOK, gin.H{"bank_accounts": accounts})
}

func CreateBankAccount(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}

	var req models.CreateBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IsPrimary {
		database.DB.Exec("UPDATE toko_bank_accounts SET is_primary = FALSE WHERE toko_id = ?", tokoID)
	}

	id := uuid.New().String()
	now := time.Now()
	_, err := database.DB.Exec(
		`INSERT INTO toko_bank_accounts (id, toko_id, bank_name, bank_code, account_number, account_holder_name, is_primary, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tokoID, req.BankName, req.BankCode, req.AccountNumber, req.AccountHolderName, req.IsPrimary, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bank account"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"bank_account_id": id, "message": "Bank account created"})
}

func UpdateBankAccount(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}
	accountID := c.Param("id")

	var req models.UpdateBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var owned int
	if err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM toko_bank_accounts WHERE id = ? AND toko_id = ?", accountID, tokoID,
	).Scan(&owned); err != nil || owned == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank account not found"})
		return
	}

	if req.BankName != "" {
		database.DB.Exec("UPDATE toko_bank_accounts SET bank_name = ? WHERE id = ?", req.BankName, accountID)
	}
	if req.BankCode != "" {
		database.DB.Exec("UPDATE toko_bank_accounts SET bank_code = ? WHERE id = ?", req.BankCode, accountID)
	}
	if req.AccountNumber != "" {
		database.DB.Exec("UPDATE toko_bank_accounts SET account_number = ? WHERE id = ?", req.AccountNumber, accountID)
	}
	if req.AccountHolderName != "" {
		database.DB.Exec("UPDATE toko_bank_accounts SET account_holder_name = ? WHERE id = ?", req.AccountHolderName, accountID)
	}
	if req.IsPrimary != nil && *req.IsPrimary {
		database.DB.Exec("UPDATE toko_bank_accounts SET is_primary = FALSE WHERE toko_id = ?", tokoID)
		database.DB.Exec("UPDATE toko_bank_accounts SET is_primary = TRUE WHERE id = ?", accountID)
	}
	database.DB.Exec("UPDATE toko_bank_accounts SET updated_at = ? WHERE id = ?", time.Now(), accountID)

	c.JSON(http.StatusOK, gin.H{"message": "Bank account updated"})
}

func DeleteBankAccount(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}
	accountID := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM toko_bank_accounts WHERE id = ? AND toko_id = ?", accountID, tokoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bank account"})
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank account not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Bank account deleted"})
}

func SetPrimaryBankAccount(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}
	accountID := c.Param("id")

	var owned int
	if err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM toko_bank_accounts WHERE id = ? AND toko_id = ?", accountID, tokoID,
	).Scan(&owned); err != nil || owned == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank account not found"})
		return
	}

	database.DB.Exec("UPDATE toko_bank_accounts SET is_primary = FALSE WHERE toko_id = ?", tokoID)
	database.DB.Exec("UPDATE toko_bank_accounts SET is_primary = TRUE WHERE id = ?", accountID)

	c.JSON(http.StatusOK, gin.H{"message": "Primary bank account updated"})
}

// ──────────────── API KEY POS ────────────────

func GetMyApiKeys(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, toko_id, label, api_key, is_active, last_used_at, created_at
		 FROM toko_api_keys WHERE toko_id = ? ORDER BY created_at DESC`, tokoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var keys []models.TokoApiKey
	for rows.Next() {
		var k models.TokoApiKey
		rows.Scan(&k.ID, &k.TokoID, &k.Label, &k.APIKey, &k.IsActive, &k.LastUsedAt, &k.CreatedAt)
		keys = append(keys, k)
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

func CreateApiKey(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}

	var req models.CreateApiKeyRequest
	_ = c.ShouldBindJSON(&req)

	id := uuid.New().String()
	key := randomKey(32)
	_, err := database.DB.Exec(
		`INSERT INTO toko_api_keys (id, toko_id, label, api_key, is_active, created_at)
		 VALUES (?, ?, ?, ?, TRUE, ?)`,
		id, tokoID, req.Label, key, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"api_key_id": id, "api_key": key, "message": "API key created"})
}

func RevokeApiKey(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}
	keyID := c.Param("id")

	_, err := database.DB.Exec(
		"UPDATE toko_api_keys SET is_active = FALSE WHERE id = ? AND toko_id = ?", keyID, tokoID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

func RotateApiKey(c *gin.Context) {
	tokoID, ok := requireTokoID(c)
	if !ok {
		return
	}
	keyID := c.Param("id")

	var owned int
	if err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM toko_api_keys WHERE id = ? AND toko_id = ?", keyID, tokoID,
	).Scan(&owned); err != nil || owned == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	key := randomKey(32)
	if _, err := database.DB.Exec(
		"UPDATE toko_api_keys SET api_key = ?, is_active = TRUE WHERE id = ?", key, keyID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rotate API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": key, "message": "API key rotated"})
}

// ──────────────── SERTIFIKAT MITRA HIJAU ────────────────

func GetMyCertificate(c *gin.Context) {
	userID := c.GetString("user_id")

	var tp models.TokoProfile
	err := database.DB.QueryRow(
		`SELECT id, user_id, business_name, business_category, address, latitude, longitude,
		        legal_document_url, operational_hours, verification_status, verified_by_admin_id,
		        verified_at, average_rating, created_at, updated_at
		 FROM toko_profiles WHERE user_id = ?`, userID,
	).Scan(
		&tp.ID, &tp.UserID, &tp.BusinessName, &tp.BusinessCategory,
		&tp.Address, &tp.Latitude, &tp.Longitude, &tp.LegalDocumentURL,
		&tp.OperationalHours, &tp.VerificationStatus, &tp.VerifiedByAdminID,
		&tp.VerifiedAt, &tp.AverageRating, &tp.CreatedAt, &tp.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return
	}

	var totalOrders int
	var totalKg, totalRevenue float64
	database.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(o.quantity * 0.5),0), COALESCE(SUM(o.total_amount),0)
		FROM orders o JOIN food_listings fl ON o.listing_id = fl.id
		WHERE fl.toko_id = ? AND o.order_status = 'selesai'`, tp.ID).Scan(&totalOrders, &totalKg, &totalRevenue)

	var communityHandovers int
	database.DB.QueryRow("SELECT COUNT(*) FROM community_posts WHERE toko_id = ? AND claim_status = 'selesai'", tp.ID).Scan(&communityHandovers)

	issuedAt := tp.CreatedAt
	if tp.VerifiedAt.Valid {
		issuedAt = tp.VerifiedAt.Time
	}

	c.JSON(http.StatusOK, gin.H{
		"certificate": gin.H{
			"toko_id":             tp.ID,
			"business_name":       tp.BusinessName,
			"business_category":   tp.BusinessCategory,
			"verification_status": tp.VerificationStatus,
			"issued_at":           issuedAt,
			"total_food_saved_kg": totalKg,
			"total_orders":        totalOrders,
			"total_revenue":       totalRevenue,
			"estimated_co2_kg":    totalKg * 2.5,
			"community_handovers": communityHandovers,
			"certificate_number":  "FR-MH-" + strings.ToUpper(tp.ID[:8]),
		},
	})
}

// ──────────────── HELPERS ────────────────

func requireTokoID(c *gin.Context) (string, bool) {
	userID := c.GetString("user_id")
	var tokoID string
	err := database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ?", userID).Scan(&tokoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Toko profile not found"})
		return "", false
	}
	return tokoID, true
}

func randomKey(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
