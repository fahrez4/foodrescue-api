package routes

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "foodrescue-api/docs"
	"foodrescue-api/internal/middleware"
	"foodrescue-api/internal/handlers/auth"
	"foodrescue-api/internal/handlers/user"
	"foodrescue-api/internal/handlers/toko"
	"foodrescue-api/internal/handlers/kurir"
	"foodrescue-api/internal/handlers/listing"
	"foodrescue-api/internal/handlers/order"
	"foodrescue-api/internal/handlers/delivery"
	"foodrescue-api/internal/handlers/community"
	"foodrescue-api/internal/handlers/emergency"
	"foodrescue-api/internal/handlers/admin"
	"foodrescue-api/internal/handlers/rating"
	"foodrescue-api/internal/handlers/report"
	"foodrescue-api/internal/handlers/chat"
	"foodrescue-api/internal/handlers/ai"
	"foodrescue-api/internal/handlers/payment"
)

func Setup(r *gin.Engine) {
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		// Health check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "service": "food-rescue-api"})
		})

		// ───────── AUTH ─────────
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", auth.Register)
			authGroup.POST("/login", auth.Login)
			authGroup.POST("/google", auth.GoogleLogin)
			authGroup.POST("/google/register", auth.GoogleRegister)

			authed := authGroup.Group("", middleware.AuthMiddleware())
			{
				authed.GET("/profile", auth.GetProfile)
				authed.PUT("/profile", auth.UpdateProfile)
			}
		}

		// ───────── USER ─────────
		userGroup := api.Group("/users")
		{
			userGroup.GET("/:id", user.GetUser)

			authed := userGroup.Group("", middleware.AuthMiddleware(), middleware.RoleGuard("user", "toko", "kurir", "admin"))
			{
				authed.GET("/me/impact", user.GetMyImpact)
				authed.POST("/payment-methods", user.RegisterPaymentMethod)
				authed.GET("/payment-methods", user.ListPaymentMethods)
			}
		}

		// ───────── TOKO ─────────
		tokoGroup := api.Group("/tokos")
		{
			tokoGroup.GET("", toko.ListApprovedTokos)
			tokoGroup.GET("/:id", toko.GetTokoByID)

			authed := tokoGroup.Group("", middleware.AuthMiddleware(), middleware.RoleGuard("toko"))
			{
				authed.POST("", toko.CreateTokoProfile)
				authed.GET("/me/profile", toko.GetMyTokoProfile)
				authed.PUT("/me/profile", toko.UpdateTokoProfile)
				authed.GET("/me/analytics", toko.GetSalesAnalytics)
				authed.GET("/me/ratings", toko.GetTokoRatings)
			}
		}

		// ───────── KURIR ─────────
		kurirGroup := api.Group("/kurirs")
		{
			authed := kurirGroup.Group("", middleware.AuthMiddleware(), middleware.RoleGuard("kurir"))
			{
				authed.GET("/me/profile", kurir.GetMyCourierProfile)
				authed.PUT("/me/profile", kurir.UpdateCourierProfile)
				authed.PATCH("/me/online", kurir.ToggleOnline)
				authed.PUT("/me/location", kurir.UpdateLocation)
				authed.GET("/deliveries/pending", kurir.GetPendingDeliveries)
				authed.POST("/deliveries/accept", kurir.AcceptDelivery)
				authed.PUT("/deliveries/trip", kurir.UpdateTripStatus)
				authed.POST("/deliveries/confirm-pickup", kurir.ConfirmPickup)
				authed.POST("/deliveries/confirm-dropoff", kurir.ConfirmDropoff)
				authed.GET("/me/earnings", kurir.GetCourierEarnings)
			}
		}

		// ───────── LISTINGS ─────────
		listingGroup := api.Group("/listings")
		{
			listingGroup.GET("", listing.ListListings)
			listingGroup.GET("/:id", listing.GetListing)

			authed := listingGroup.Group("", middleware.AuthMiddleware())
			{
				authed.POST("", middleware.RoleGuard("toko"), listing.CreateListing)
				authed.GET("/me/listings", middleware.RoleGuard("toko"), listing.GetMyListings)
				authed.PUT("/:id", middleware.RoleGuard("toko"), listing.UpdateListing)
				authed.DELETE("/:id", middleware.RoleGuard("toko"), listing.DeleteListing)
			}
		}

		// ───────── ORDERS ─────────
		orderGroup := api.Group("/orders")
		{
			authed := orderGroup.Group("", middleware.AuthMiddleware())
			{
				authed.POST("", middleware.RoleGuard("user"), order.CreateOrder)
				authed.GET("/me", order.GetMyOrders)
				authed.GET("/:id", order.GetOrder)
				authed.POST("/:id/cancel", order.CancelOrder)
				authed.POST("/:id/confirm-payment", order.ConfirmPayment)
			}
		}

		// ───────── DELIVERIES ─────────
		deliveryGroup := api.Group("/deliveries")
		{
			authed := deliveryGroup.Group("", middleware.AuthMiddleware())
			{
				authed.GET("/order/:order_id", delivery.GetDeliveryByOrder)
				authed.GET("/me", delivery.GetMyDeliveries)
			}
		}

		// ───────── COMMUNITY ─────────
		communityGroup := api.Group("/community")
		{
			communityGroup.GET("", community.ListCommunityPosts)

			authed := communityGroup.Group("", middleware.AuthMiddleware())
			{
				authed.POST("", middleware.RoleGuard("toko"), community.CreateCommunityPost)
				authed.POST("/claim", middleware.RoleGuard("user"), community.ClaimCommunityPost)
				authed.POST("/:id/confirm", community.ConfirmCommunityPickup)
			}
		}

		// ───────── EMERGENCY ─────────
		emergencyGroup := api.Group("/emergency")
		{
			emergencyGroup.GET("/alerts", emergency.ListActiveAlerts)
			emergencyGroup.GET("/alerts/:id/responses", emergency.GetAlertResponses)

			authed := emergencyGroup.Group("", middleware.AuthMiddleware())
			{
				authed.POST("/alerts", middleware.RoleGuard("user"), emergency.CreateAlert)
				authed.POST("/responses", middleware.RoleGuard("toko"), emergency.RespondToAlert)
			}
		}

		// ───────── RATINGS ─────────
		ratingGroup := api.Group("/ratings")
		{
			ratingGroup.GET("/:id", rating.GetRatingsForUser)

			authed := ratingGroup.Group("", middleware.AuthMiddleware())
			{
				authed.POST("", rating.CreateRating)
				authed.GET("/me/given", rating.GetMyGivenRatings)
			}
		}

		// ───────── REPORTS ─────────
		reportGroup := api.Group("/reports")
		{
			authed := reportGroup.Group("", middleware.AuthMiddleware())
			{
				authed.POST("", report.CreateReport)
				authed.GET("/me", report.GetMyReports)
			}
		}

		// ───────── CHAT ─────────
		chatGroup := api.Group("/chats")
		{
			authed := chatGroup.Group("", middleware.AuthMiddleware())
			{
				authed.POST("", chat.CreateChat)
				authed.GET("/me", chat.GetMyChats)
				authed.GET("/:id/messages", chat.GetMessages)
				authed.POST("/:id/messages", chat.SendMessage)
			}
		}

		// ───────── AI ─────────
		aiGroup := api.Group("/ai")
		{
			authed := aiGroup.Group("", middleware.AuthMiddleware(), middleware.RoleGuard("user"))
			{
				authed.POST("/chat", ai.ChatFromListing)
				authed.POST("/detect", ai.DetectFromCamera)
				authed.GET("/conversations/:id/messages", ai.GetConversationMessages)
				authed.POST("/conversations/:id/messages", ai.ContinueChat)
			}
		}

		// ───────── PAYMENT ─────────
		paymentGroup := api.Group("/payments")
		{
			paymentGroup.POST("/create", middleware.AuthMiddleware(), payment.CreateTransaction)
			paymentGroup.POST("/notification", payment.HandleNotification)
		}

		// ───────── ADMIN ─────────
		adminGroup := api.Group("/admin")
		{
			authed := adminGroup.Group("", middleware.AuthMiddleware(), middleware.RoleGuard("admin"))
			{
				authed.GET("/dashboard", admin.GetDashboardAnalytics)
				authed.GET("/users", admin.ListUsers)
				authed.GET("/verifications", admin.ListPendingVerifications)
				authed.POST("/verifications/:id", admin.VerifyUser)
				authed.DELETE("/users/:id", admin.DeleteUser)
				authed.PUT("/users/:id/deactivate", admin.DeactivateUser)
				authed.PUT("/users/:id/verify-ngo", admin.VerifyNGO)
				authed.GET("/reports", admin.ListReports)
				authed.PUT("/reports/:id", admin.ReviewReport)
			}
		}
	}
}