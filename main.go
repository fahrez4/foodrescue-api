package main

import (
	"log"
	"net/http"
	"os"

	"foodrescue-api/config"
	"foodrescue-api/controllers"
	_ "foodrescue-api/docs"
	"foodrescue-api/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" 
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Catatan: File .env tidak ditemukan, menggunakan environment variable bawaan sistem.")
	}

	db := config.ConnectDB()
	defer db.Close()

	r := gin.Default()

	r.Use(CORSMiddleware())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"app":     "FoodRescue REST API",
			"status":  "running",
			"version": "1.0.0",
			"docs":    "http://localhost:8080/swagger/index.html",
		})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		v1.POST("/register", controllers.Register)
		v1.POST("/login", controllers.Login)
		v1.GET("/foods", controllers.GetFoods)          
		v1.GET("/foods/:id", controllers.GetFoodByID)   

		protected := v1.Group("")
		protected.Use(middlewares.AuthMiddleware())
		{
			protected.GET("/my-foods", controllers.GetMyFoods)        
			protected.POST("/foods", controllers.CreateFood)          
			protected.PUT("/foods/:id", controllers.UpdateFood)       
			protected.DELETE("/foods/:id", controllers.DeleteFood)    

			protected.POST("/foods/:id/claim", controllers.ClaimFood) 
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8091"
	}

	log.Printf("Server FoodRescue API berjalan di http://localhost:%s\n", port)
	log.Printf("Dokumentasi Swagger UI tersedia di http://localhost:%s/swagger/index.html\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}