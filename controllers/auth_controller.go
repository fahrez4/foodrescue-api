package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"foodrescue-api/config"
	"foodrescue-api/middlewares"
	"foodrescue-api/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Name     string `json:"name" binding:"required" example:"Budi Santoso"`
	Email    string `json:"email" binding:"required,email" example:"budi@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"rahasia123"`
	Phone    string `json:"phone" example:"081234567890"`
	Address  string `json:"address" example:"Jl. Sudirman No. 45, Jakarta"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"budi@example.com"`
	Password string `json:"password" binding:"required" example:"rahasia123"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("Data input tidak valid: "+err.Error()))
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var existingID string
	err := config.DB.QueryRow("SELECT id FROM users WHERE email = ?", req.Email).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, models.BuildErrorResponse("Email sudah terdaftar"))
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Terjadi kesalahan pada database"))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal melakukan hashing password"))
		return
	}

	userID := uuid.New().String()

	query := `
		INSERT INTO users (id, name, email, password_hash, phone, address, created_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW())
	`
	_, err = config.DB.Exec(query, userID, req.Name, req.Email, string(hashedPassword), req.Phone, req.Address)
	if err != nil {
		log.Printf("[Error] Register DB Exec: %v", err)
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal menyimpan pengguna ke database"))
		return
	}

	userData := gin.H{
		"id":      userID,
		"name":    req.Name,
		"email":   req.Email,
		"phone":   req.Phone,
		"address": req.Address,
	}

	c.JSON(http.StatusCreated, models.BuildResponse(true, "Registrasi berhasil", userData))
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.BuildErrorResponse("Data login tidak valid: "+err.Error()))
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var (
		id           string
		name         string
		email        string
		passwordHash string
		phone        sql.NullString
		address      sql.NullString
	)

	query := "SELECT id, name, email, password_hash, phone, address FROM users WHERE email = ?"
	err := config.DB.QueryRow(query, req.Email).Scan(&id, &name, &email, &passwordHash, &phone, &address)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, models.BuildErrorResponse("Email atau kata sandi tidak cocok"))
			return
		}
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal memproses data login"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, models.BuildErrorResponse("Email atau kata sandi tidak cocok"))
		return
	}

	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	claims := middlewares.JWTClaims{
		UserID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "foodrescue-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middlewares.GetJWTSecret())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.BuildErrorResponse("Gagal membuat token autentikasi"))
		return
	}

	responseData := gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":      id,
			"name":    name,
			"email":   email,
			"phone":   phone.String,
			"address": address.String,
		},
	}

	c.JSON(http.StatusOK, models.BuildResponse(true, "Login berhasil", responseData))
}
