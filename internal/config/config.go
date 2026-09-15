package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                  string
	AppPort                 string
	DBHost                  string
	DBPort                  string
	DBUser                  string
	DBPassword              string
	DBName                  string
	DBDSN                   string // optional full DSN override (digunakan Docker)
	JWTSecret               string
	GoogleClientID          string
	GeminiAPIKey            string
	PaymentGatewayKey       string
	PaymentGatewayServerKey string
	PaymentGatewayClientKey string
}

var AppConfig Config

func Load() {
	godotenv.Load()

	dsn := getEnv("DB_DSN", "")

	AppConfig = Config{
		AppEnv:                  getEnv("APP_ENV", "development"),
		AppPort:                 getEnv("APP_PORT", "8080"),
		DBHost:                  getEnv("DB_HOST", "localhost"),
		DBPort:                  getEnv("DB_PORT", "3306"),
		DBUser:                  getEnv("DB_USER", "root"),
		DBPassword:              getEnv("DB_PASSWORD", ""),
		DBName:                  getEnv("DB_NAME", "food_rescue"),
		DBDSN:                   dsn,
		JWTSecret:               getEnv("JWT_SECRET", ""),
		GoogleClientID:          getEnv("GOOGLE_CLIENT_ID", ""),
		GeminiAPIKey:            getEnv("GEMINI_API_KEY", ""),
		PaymentGatewayKey:       getEnv("PAYMENT_GATEWAY_SERVER_KEY", ""),
		PaymentGatewayServerKey: getEnv("MB_SERVER_KEY", ""),
		PaymentGatewayClientKey: getEnv("MB_CLIENT_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}