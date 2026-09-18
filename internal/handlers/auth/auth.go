package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v4"

	"foodrescue-api/internal/config"
	"foodrescue-api/internal/database"
	"foodrescue-api/internal/middleware"
	"foodrescue-api/internal/models"
)

func Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var exists int
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", req.Email).Scan(&exists)
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	id := uuid.New().String()
	now := time.Now()
	accountStatus := "active"
	if req.Role == "toko" || req.Role == "kurir" {
		accountStatus = "pending_verification"
	}

	var passwordHash models.NullString
	if req.Password != "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		passwordHash = models.NullString{String: string(hash), Valid: true}
	}

	authProvider := "email_password"
	if req.Password == "" {
		authProvider = "google"
	}

	_, err := database.DB.Exec(
		`INSERT INTO users (id, email, full_name, phone_number, auth_provider, role, trust_score, account_status, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 5.00, ?, ?, ?, ?)`,
		id, req.Email, req.FullName, models.NullString{String: req.PhoneNumber, Valid: req.PhoneNumber != ""},
		authProvider, req.Role, accountStatus, passwordHash, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	if req.Role == "toko" {
		tokoID := uuid.New().String()
		database.DB.Exec(
			`INSERT INTO toko_profiles (id, user_id, business_name, business_category, address, verification_status, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, 'pending', ?, ?)`,
			tokoID, id, "", "", "", now, now,
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

	token := generateToken(id, req.Email, req.Role)

	user := models.User{
		ID:            id,
		Email:         req.Email,
		FullName:      req.FullName,
		AuthProvider:  authProvider,
		Role:          req.Role,
		TrustScore:    5.00,
		AccountStatus: accountStatus,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	c.JSON(http.StatusCreated, models.AuthResponse{Token: token, User: user})
}

func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	var passwordHash models.NullString
	err := database.DB.QueryRow(
		`SELECT id, email, full_name, photo_url, phone_number, auth_provider, role, is_ngo_verified,
		        trust_score, latitude, longitude, address_text, account_status, password_hash, created_at, updated_at
		 FROM users WHERE email = ?`, req.Email,
	).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PhotoURL, &user.PhoneNumber,
		&user.AuthProvider, &user.Role, &user.IsNGOVerified, &user.TrustScore,
		&user.Latitude, &user.Longitude, &user.AddressText, &user.AccountStatus,
		&passwordHash, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !passwordHash.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "This account uses Google Sign-In. Please login with Google."})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash.String), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if user.AccountStatus == "suspended" || user.AccountStatus == "rejected" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Akun Anda telah diblokir. Silakan hubungi admin.",
			"code":  "ACCOUNT_BLOCKED",
		})
		return
	}

	token := generateToken(user.ID, user.Email, user.Role)
	c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: user})
}

func GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var user models.User
	err := database.DB.QueryRow(
		`SELECT id, email, full_name, photo_url, phone_number, auth_provider, role, is_ngo_verified,
		        trust_score, latitude, longitude, address_text, account_status, created_at, updated_at
		 FROM users WHERE id = ?`, userID,
	).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PhotoURL, &user.PhoneNumber,
		&user.AuthProvider, &user.Role, &user.IsNGOVerified, &user.TrustScore,
		&user.Latitude, &user.Longitude, &user.AddressText, &user.AccountStatus,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FullName != "" {
		database.DB.Exec("UPDATE users SET full_name = ? WHERE id = ?", req.FullName, userID)
	}
	if req.PhoneNumber != "" {
		database.DB.Exec("UPDATE users SET phone_number = ? WHERE id = ?", req.PhoneNumber, userID)
	}
	if req.PhotoURL != "" {
		database.DB.Exec("UPDATE users SET photo_url = ? WHERE id = ?", req.PhotoURL, userID)
	}
	if req.AddressText != "" {
		database.DB.Exec("UPDATE users SET address_text = ?, latitude = ?, longitude = ? WHERE id = ?",
			req.AddressText, req.Latitude, req.Longitude, userID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// SwitchRole — pindah role aktif (multi-role) tanpa login ulang.
// Eliglible: user selalu boleh jadi 'user'; jadi toko/kurir hanya jika profil ada.
func SwitchRole(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.SwitchRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role == "toko" {
		var n int
		_ = database.DB.QueryRow("SELECT COUNT(*) FROM toko_profiles WHERE user_id = ?", userID).Scan(&n)
		if n == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun ini tidak memiliki profil Mitra Toko"})
			return
		}
	}
	if req.Role == "kurir" {
		var n int
		_ = database.DB.QueryRow("SELECT COUNT(*) FROM courier_profiles WHERE user_id = ?", userID).Scan(&n)
		if n == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun ini tidak memiliki profil Kurir"})
			return
		}
	}

	if _, err := database.DB.Exec(
		"UPDATE users SET role = ?, updated_at = ? WHERE id = ?", req.Role, time.Now(), userID,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to switch role"})
		return
	}

	var user models.User
	err := database.DB.QueryRow(
		`SELECT id, email, full_name, photo_url, phone_number, auth_provider, role, is_ngo_verified,
		        trust_score, latitude, longitude, address_text, account_status, created_at, updated_at
		 FROM users WHERE id = ?`, userID,
	).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PhotoURL, &user.PhoneNumber,
		&user.AuthProvider, &user.Role, &user.IsNGOVerified, &user.TrustScore,
		&user.Latitude, &user.Longitude, &user.AddressText, &user.AccountStatus,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload user"})
		return
	}

	token := generateToken(user.ID, user.Email, user.Role)
	c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: user})
}

func generateToken(userID, email, role string) string {
	claims := &middleware.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(config.AppConfig.JWTSecret))
	return tokenString
}
