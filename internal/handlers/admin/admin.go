package admin

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

// adminName mengambil nama admin untuk jejak audit.
func adminName(adminID string) string {
	var name string
	_ = database.DB.QueryRow("SELECT full_name FROM users WHERE id = ?", adminID).Scan(&name)
	return name
}

// logAdminAction menyimpan satu baris jejak audit aksi admin.
func logAdminAction(adminID, action, targetType, targetID, description string) {
	if adminID == "" {
		return
	}
	_, err := database.DB.Exec(
		`INSERT INTO admin_activity_logs
		 (id, admin_user_id, admin_name, action, target_type, target_id, description, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), adminID, adminName(adminID), action, targetType, targetID, description, time.Now(),
	)
	if err != nil {
		log.Printf("admin activity log failed: %v", err)
	}
}

func ListPendingVerifications(c *gin.Context) {
	type PendingUser struct {
		models.User
		BusinessName     models.NullString `json:"business_name"`
		BusinessCategory models.NullString `json:"business_category"`
		LegalDocumentURL models.NullString `json:"legal_document_url"`
		VehicleType      models.NullString `json:"vehicle_type"`
		IDDocumentURL    models.NullString `json:"id_document_url"`
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

	logAdminAction(adminID, "verify_user", "user", targetID, "Verifikasi user: "+req.Status)
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
	adminID := c.GetString("user_id")
	targetID := c.Param("id")
	_, err := database.DB.Exec("UPDATE users SET account_status = 'suspended' WHERE id = ?", targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
		return
	}
	logAdminAction(adminID, "deactivate_user", "user", targetID, "Blokir/suspend akun")
	c.JSON(http.StatusOK, gin.H{"message": "User deactivated"})
}

func DeleteUser(c *gin.Context) {
	adminID := c.GetString("user_id")
	targetID := c.Param("id")

	var targetEmail string
	_ = database.DB.QueryRow("SELECT email FROM users WHERE id = ?", targetID).Scan(&targetEmail)

	_, err := database.DB.Exec("DELETE FROM users WHERE id = ?", targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	logAdminAction(adminID, "delete_user", "user", targetID, "Hapus akun: "+targetEmail)
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

	logAdminAction(adminID, "review_report", "report", reportID, "Status laporan: "+req.Status)
	c.JSON(http.StatusOK, gin.H{"message": "Report " + req.Status})
}

func VerifyNGO(c *gin.Context) {
	adminID := c.GetString("user_id")
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

	status := "dicabut"
	if req.IsVerified {
		status = "diverifikasi"
	}
	logAdminAction(adminID, "verify_ngo", "user", targetID, "Status NGO: "+status)
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

// AdminOrderView — daftar pesanan lintas platform untuk panel admin.
type AdminOrderView struct {
	models.Order
	BuyerName   string `json:"buyer_name"`
	BuyerEmail  string `json:"buyer_email"`
	ListingName string `json:"listing_name"`
	TokoName    string `json:"toko_name"`
}

// ListOrders — GET /admin/orders (filter status & pencarian opsional).
func ListOrders(c *gin.Context) {
	status := c.Query("status")
	q := strings.TrimSpace(c.Query("q"))

	query := `SELECT o.id, o.listing_id, o.user_id, o.quantity, o.price_at_purchase, o.total_amount,
	                 o.fulfillment_method, COALESCE(o.payment_method,''), o.payment_status, o.order_status,
	                 o.confirmation_code, o.created_at, o.completed_at,
	                 COALESCE(u.full_name,''), COALESCE(u.email,''),
	                 COALESCE(fl.name,''), COALESCE(tp.business_name,'')
	          FROM orders o
	          LEFT JOIN users u ON o.user_id = u.id
	          LEFT JOIN food_listings fl ON o.listing_id = fl.id
	          LEFT JOIN toko_profiles tp ON fl.toko_id = tp.id
	          WHERE 1=1`
	var args []interface{}

	if status != "" {
		query += " AND o.order_status = ?"
		args = append(args, status)
	}
	if q != "" {
		like := "%" + q + "%"
		query += " AND (u.full_name LIKE ? OR u.email LIKE ? OR fl.name LIKE ? OR tp.business_name LIKE ? OR o.confirmation_code LIKE ?)"
		args = append(args, like, like, like, like, like)
	}
	query += " ORDER BY o.created_at DESC LIMIT 500"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	orders := make([]AdminOrderView, 0)
	for rows.Next() {
		var o AdminOrderView
		if err := rows.Scan(&o.ID, &o.ListingID, &o.UserID, &o.Quantity, &o.PriceAtPurchase, &o.TotalAmount,
			&o.FulfillmentMethod, &o.PaymentMethod, &o.PaymentStatus, &o.OrderStatus,
			&o.ConfirmationCode, &o.CreatedAt, &o.CompletedAt,
			&o.BuyerName, &o.BuyerEmail, &o.ListingName, &o.TokoName); err != nil {
			continue
		}
		orders = append(orders, o)
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders, "total": len(orders)})
}

// UpdateOrderStatus — PUT /admin/orders/:id/status (override status oleh admin).
func UpdateOrderStatus(c *gin.Context) {
	adminID := c.GetString("user_id")
	orderID := c.Param("id")

	var req struct {
		OrderStatus string `json:"order_status" binding:"required,oneof=menunggu_pickup selesai dibatalkan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var listingID string
	var quantity int
	if err := database.DB.QueryRow(
		"SELECT listing_id, quantity FROM orders WHERE id = ?", orderID,
	).Scan(&listingID, &quantity); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	switch req.OrderStatus {
	case "selesai":
		_, err := database.DB.Exec(
			"UPDATE orders SET order_status = 'selesai', payment_status = 'paid', completed_at = ? WHERE id = ?",
			time.Now(), orderID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
			return
		}
	case "dibatalkan":
		if _, err := database.DB.Exec("UPDATE orders SET order_status = 'dibatalkan' WHERE id = ?", orderID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
			return
		}
		database.DB.Exec(
			"UPDATE food_listings SET stock_quantity = stock_quantity + ?, status = CASE WHEN status = 'sold_out' THEN 'active' ELSE status END WHERE id = ?",
			quantity, listingID,
		)
	default:
		if _, err := database.DB.Exec("UPDATE orders SET order_status = ?, completed_at = NULL WHERE id = ?", req.OrderStatus, orderID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
			return
		}
	}

	logAdminAction(adminID, "update_order_status", "order", orderID, "Status pesanan → "+req.OrderStatus)
	c.JSON(http.StatusOK, gin.H{"message": "Order status updated", "order_status": req.OrderStatus})
}

// AdminLogView — satu baris jejak audit admin.
type AdminLogView struct {
	ID          string    `json:"id"`
	AdminUserID string    `json:"admin_user_id"`
	AdminName   string    `json:"admin_name"`
	Action      string    `json:"action"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListLogs — GET /admin/logs.
func ListLogs(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, admin_user_id, admin_name, action, target_type, target_id, description, created_at
		 FROM admin_activity_logs ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	logs := make([]AdminLogView, 0)
	for rows.Next() {
		var l AdminLogView
		if err := rows.Scan(&l.ID, &l.AdminUserID, &l.AdminName, &l.Action,
			&l.TargetType, &l.TargetID, &l.Description, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
