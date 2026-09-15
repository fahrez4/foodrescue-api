package admin

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func ListPendingVerifications(c *gin.Context) {
	type PendingUser struct {
		models.User
		BusinessName      models.NullString `json:"business_name"`
		BusinessCategory  models.NullString `json:"business_category"`
		LegalDocumentURL  models.NullString `json:"legal_document_url"`
		VehicleType       models.NullString `json:"vehicle_type"`
		IDDocumentURL     models.NullString `json:"id_document_url"`
	}

	rows, err := database.DB.Query(
		`SELECT u.id, u.email, u.full_name, u.photo_url, u.phone_number, u.auth_provider, u.role,
		        u.is_ngo_verified, u.trust_score, u.latitude, u.longitude, u.address_text,
		        u.account_status, u.created_at, u.updated_at,
		        tp.business_name, tp.business_category, tp.legal_document_url,
		        cp.vehicle_type, cp.id_document_url
		 FROM users u
		 LEFT JOIN toko_profiles tp ON u.id = tp.user_id AND tp.verification_status = 'pending'
		 LEFT JOIN courier_profiles cp ON u.id = cp.user_id AND cp.verification_status = 'pending'
		 WHERE u.account_status = 'pending_verification'`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var users []PendingUser
	for rows.Next() {
		var u PendingUser
		rows.Scan(&u.ID, &u.Email, &u.FullName, &u.PhotoURL, &u.PhoneNumber,
			&u.AuthProvider, &u.Role, &u.IsNGOVerified, &u.TrustScore,
			&u.Latitude, &u.Longitude, &u.AddressText, &u.AccountStatus,
			&u.CreatedAt, &u.UpdatedAt, &u.BusinessName, &u.BusinessCategory,
			&u.LegalDocumentURL, &u.VehicleType, &u.IDDocumentURL)
		users = append(users, u)
	}

	c.JSON(http.StatusOK, gin.H{"pending_users": users})
}

func VerifyUser(c *gin.Context) {
	adminID := c.GetString("user_id")
	targetID := c.Param("id")

	var req models.AdminVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userStatus := "active"
	if req.Status == "rejected" {
		userStatus = "rejected"
	}

	_, err := database.DB.Exec("UPDATE users SET account_status = ? WHERE id = ?", userStatus, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	now := time.Now()
	if req.Status == "approved" {
		database.DB.Exec("UPDATE toko_profiles SET verification_status = 'approved', verified_by_admin_id = ?, verified_at = ? WHERE user_id = ?",
			adminID, now, targetID)
		database.DB.Exec("UPDATE courier_profiles SET verification_status = 'approved', verified_by_admin_id = ?, verified_at = ? WHERE user_id = ?",
			adminID, now, targetID)
	} else {
		database.DB.Exec("UPDATE toko_profiles SET verification_status = 'rejected', verified_by_admin_id = ?, verified_at = ? WHERE user_id = ?",
			adminID, now, targetID)
		database.DB.Exec("UPDATE courier_profiles SET verification_status = 'rejected', verified_by_admin_id = ?, verified_at = ? WHERE user_id = ?",
			adminID, now, targetID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "User " + req.Status + " successfully"})
}

func ListUsers(c *gin.Context) {
	role := c.Query("role")

	query := `SELECT id, email, full_name, role, account_status, trust_score, created_at FROM users`
	var args []interface{}

	if role != "" {
		query += " WHERE role = ?"
		args = append(args, role)
	}
	query += " ORDER BY created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type UserSummary struct {
		ID            string    `json:"id"`
		Email         string    `json:"email"`
		FullName      string    `json:"full_name"`
		Role          string    `json:"role"`
		AccountStatus string    `json:"account_status"`
		TrustScore    float64   `json:"trust_score"`
		CreatedAt     time.Time `json:"created_at"`
	}

	var users []UserSummary
	for rows.Next() {
		var u UserSummary
		rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.AccountStatus, &u.TrustScore, &u.CreatedAt)
		users = append(users, u)
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func DeactivateUser(c *gin.Context) {
	targetID := c.Param("id")
	_, err := database.DB.Exec("UPDATE users SET account_status = 'suspended' WHERE id = ?", targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deactivated"})
}

func DeleteUser(c *gin.Context) {
	targetID := c.Param("id")
	_, err := database.DB.Exec("DELETE FROM users WHERE id = ?", targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func ListReports(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT r.id, r.reporter_user_id, r.reported_entity_type, r.reported_entity_id,
		        r.reason, r.report_status, r.reviewed_by_admin_id, r.reviewed_at, r.created_at
		 FROM reports r ORDER BY r.created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		var r models.Report
		rows.Scan(&r.ID, &r.ReporterUserID, &r.ReportedEntityType, &r.ReportedEntityID,
			&r.Reason, &r.ReportStatus, &r.ReviewedByAdminID, &r.ReviewedAt, &r.CreatedAt)
		reports = append(reports, r)
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

func ReviewReport(c *gin.Context) {
	adminID := c.GetString("user_id")
	reportID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required,oneof=ditinjau ditindak ditolak"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.DB.Exec(
		"UPDATE reports SET report_status = ?, reviewed_by_admin_id = ?, reviewed_at = ? WHERE id = ?",
		req.Status, adminID, time.Now(), reportID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report " + req.Status})
}

func VerifyNGO(c *gin.Context) {
	targetID := c.Param("id")

	var req struct {
		IsVerified bool `json:"is_verified"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.DB.Exec("UPDATE users SET is_ngo_verified = ? WHERE id = ?", req.IsVerified, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update NGO status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "NGO verification updated"})
}

func GetDashboardAnalytics(c *gin.Context) {
	var totalUsers, totalToko, totalKurir, totalAdmin int
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'user'").Scan(&totalUsers)
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'toko'").Scan(&totalToko)
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'kurir'").Scan(&totalKurir)
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&totalAdmin)

	var totalOrders, completedOrders int
	database.DB.QueryRow("SELECT COUNT(*) FROM orders").Scan(&totalOrders)
	database.DB.QueryRow("SELECT COUNT(*) FROM orders WHERE order_status = 'selesai'").Scan(&completedOrders)

	var totalRevenue float64
	database.DB.QueryRow("SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE payment_status = 'paid'").Scan(&totalRevenue)

	var totalFoodSaved float64
	database.DB.QueryRow("SELECT COALESCE(SUM(quantity * 0.5), 0) FROM orders WHERE order_status = 'selesai'").Scan(&totalFoodSaved)

	var pendingReports int
	database.DB.QueryRow("SELECT COUNT(*) FROM reports WHERE report_status = 'menunggu'").Scan(&pendingReports)

	var pendingVerifications int
	database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE account_status = 'pending_verification'").Scan(&pendingVerifications)

	c.JSON(http.StatusOK, gin.H{
		"analytics": gin.H{
			"total_users":           totalUsers,
			"total_toko":            totalToko,
			"total_kurir":           totalKurir,
			"total_admin":           totalAdmin,
			"total_orders":          totalOrders,
			"completed_orders":      completedOrders,
			"total_revenue":         totalRevenue,
			"total_food_saved_kg":   totalFoodSaved,
			"pending_reports":       pendingReports,
			"pending_verifications": pendingVerifications,
		},
	})
}
