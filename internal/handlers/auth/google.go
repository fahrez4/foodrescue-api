package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/config"
	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

const googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"

type googleTokenInfo struct {
	Iss          string `json:"iss"`
	Azp          string `json:"azp"`
	Aud          string `json:"aud"`
	Sub          string `json:"sub"`
	Email        string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name         string `json:"name"`
	Picture      string `json:"picture"`
	Exp          int64  `json:"exp"`
}

// GoogleLogin menerima id_token dari Google Sign-In (dari app mobile/web),
// memverifikasi ke endpoint tokeninfo Google, lalu login / auto-register user.
func GoogleLogin(c *gin.Context) {
	var req models.GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	info, err := verifyGoogleToken(req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token: " + err.Error()})
		return
	}

	// Validasi aud jika GoogleClientID dikonfigurasi
	if config.AppConfig.GoogleClientID != "" && info.Aud != config.AppConfig.GoogleClientID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Google token audience mismatch"})
		return
	}

	// Cek apakah user sudah terdaftar
	var userID, role, accountStatus string
	var isNGO bool
	err = database.DB.QueryRow(
		"SELECT id, role, is_ngo_verified, account_status FROM users WHERE email = ?",
		info.Email,
	).Scan(&userID, &role, &isNGO, &accountStatus)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Account not registered. Please register first, then verify with email/password or continue registration.",
			"google_profile": gin.H{
				"email":      info.Email,
				"full_name":  info.Name,
				"photo_url":  info.Picture,
				"provider":   "google",
			},
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Auto-bind Google identity jika user ada (update provider & foto)
	database.DB.Exec(
		"UPDATE users SET auth_provider = 'google', photo_url = ? WHERE id = ? AND auth_provider = 'email_password'",
		info.Picture, userID,
	)

	token := generateToken(userID, info.Email, role)

	user := models.User{
		ID:            userID,
		Email:         info.Email,
		FullName:      info.Name,
		AuthProvider:  "google",
		Role:          role,
		IsNGOVerified: isNGO,
		AccountStatus: accountStatus,
	}

	c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: user})
}

// GoogleRegister untuk user baru dengan pilihan role (dipakai app saat
// registrasi via Google Sign-In).
type GoogleRegisterRequest struct {
	IDToken     string `json:"id_token" binding:"required"`
	Role        string `json:"role" binding:"required,oneof=user toko kurir"`
	PhoneNumber string `json:"phone_number" binding:"omitempty"`
}

func GoogleRegister(c *gin.Context) {
	var req GoogleRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	info, err := verifyGoogleToken(req.IDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token: " + err.Error()})
		return
	}

	if config.AppConfig.GoogleClientID != "" && info.Aud != config.AppConfig.GoogleClientID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Google token audience mismatch"})
		return
	}

	var exists int
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", info.Email).Scan(&exists)
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered. Please login instead."})
		return
	}

	now := time.Now()
	id := uuid.New().String()
	accountStatus := "active"
	if req.Role == "toko" || req.Role == "kurir" {
		accountStatus = "pending_verification"
	}

	_, err = database.DB.Exec(
		`INSERT INTO users (id, email, full_name, photo_url, phone_number, auth_provider, role, is_ngo_verified, trust_score, account_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'google', ?, FALSE, 5.00, ?, ?, ?)`,
		id, info.Email, info.Name, info.Picture,
		models.NullString{String: req.PhoneNumber, Valid: req.PhoneNumber != ""},
		req.Role, accountStatus, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	if req.Role == "toko" {
		tokoID := uuid.New().String()
		database.DB.Exec(
			`INSERT INTO toko_profiles (id, user_id, business_name, business_category, address, verification_status, created_at, updated_at)
			 VALUES (?, ?, '', '', '', 'pending', ?, ?)`,
			tokoID, id, now, now,
		)
	}

	if req.Role == "kurir" {
		kurirID := uuid.New().String()
		database.DB.Exec(
			`INSERT INTO courier_profiles (id, user_id, vehicle_type, verification_status, created_at, updated_at)
			 VALUES (?, ?, '', 'pending', ?, ?)`,
			kurirID, id, now, now,
		)
	}

	token := generateToken(id, info.Email, req.Role)

	user := models.User{
		ID:            id,
		Email:         info.Email,
		FullName:      info.Name,
		PhotoURL:      models.NullString{String: info.Picture, Valid: info.Picture != ""},
		AuthProvider:  "google",
		Role:          req.Role,
		TrustScore:    5.00,
		AccountStatus: accountStatus,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	c.JSON(http.StatusCreated, models.AuthResponse{Token: token, User: user})
}

func verifyGoogleToken(idToken string) (*googleTokenInfo, error) {
	resp, err := http.Get(fmt.Sprintf("%s?id_token=%s", googleTokenInfoURL, idToken))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token verification failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info googleTokenInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	if info.Email == "" {
		return nil, fmt.Errorf("no email in token")
	}

	return &info, nil
}