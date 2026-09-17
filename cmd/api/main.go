package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"foodrescue-api/internal/config"
	"foodrescue-api/internal/database"
	"foodrescue-api/internal/routes"
	"foodrescue-api/internal/services"
)

func main() {
	config.Load()
	database.Connect()
	database.Migrate()
	defer database.Close()

	gin.SetMode(gin.ReleaseMode)
	if config.AppConfig.AppEnv == "development" {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	routes.Setup(router)

	// Start dynamic discount scheduler
	go services.StartDiscountScheduler()
	// Start listing expiry checker
	go services.StartExpiryScheduler()

	log.Printf("Food Rescue API running on :%s", config.AppConfig.AppPort)
	if err := router.Run(":" + config.AppConfig.AppPort); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}