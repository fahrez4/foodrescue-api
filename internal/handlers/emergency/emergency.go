package emergency

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func CreateAlert(c *gin.Context) {
	userID := c.GetString("user_id")

	var isNGO bool
	database.DB.QueryRow("SELECT is_ngo_verified FROM users WHERE id = ? AND role = 'user'", userID).Scan(&isNGO)
	if !isNGO {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only verified NGOs can create emergency alerts"})
		return
	}

	var req models.CreateEmergencyAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	_, err := database.DB.Exec(
		`INSERT INTO emergency_alerts (id, ngo_user_id, need_type, target_area, urgency_level, description, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'aktif', ?)`,
		id, userID, req.NeedType, req.TargetArea, req.UrgencyLevel, req.Description, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"alert_id": id, "message": "Emergency alert created"})
}

func ListActiveAlerts(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, ngo_user_id, need_type, target_area, urgency_level, COALESCE(description,''), status, created_at
		 FROM emergency_alerts WHERE status = 'aktif' ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var alerts []models.EmergencyAlert
	for rows.Next() {
		var a models.EmergencyAlert
		rows.Scan(&a.ID, &a.NGOUserID, &a.NeedType, &a.TargetArea, &a.UrgencyLevel, &a.Description, &a.Status, &a.CreatedAt)
		alerts = append(alerts, a)
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

func RespondToAlert(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.RespondEmergencyAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tokoID string
	err := database.DB.QueryRow("SELECT id FROM toko_profiles WHERE user_id = ? AND verification_status = 'approved'", userID).Scan(&tokoID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only verified Toko can respond to alerts"})
		return
	}

	id := uuid.New().String()
	_, err = database.DB.Exec(
		`INSERT INTO emergency_alert_responses (id, alert_id, toko_id, availability_note, response_status, created_at)
		 VALUES (?, ?, ?, ?, 'menawarkan', ?)`,
		id, req.AlertID, tokoID,
		sqlNullString(req.AvailabilityNote), time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to respond to alert"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"response_id": id, "message": "Response submitted"})
}

func GetAlertResponses(c *gin.Context) {
	alertID := c.Param("id")

	rows, err := database.DB.Query(
		`SELECT ear.id, ear.alert_id, ear.toko_id, COALESCE(ear.availability_note,''),
		        ear.response_status, ear.created_at,
		        tp.business_name
		 FROM emergency_alert_responses ear
		 JOIN toko_profiles tp ON ear.toko_id = tp.id
		 WHERE ear.alert_id = ? ORDER BY ear.created_at DESC`, alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type AlertResponseWithToko struct {
		models.EmergencyAlertResponse
		BusinessName string `json:"business_name"`
	}

	var responses []AlertResponseWithToko
	for rows.Next() {
		var r AlertResponseWithToko
		rows.Scan(&r.ID, &r.AlertID, &r.TokoID, &r.AvailabilityNote,
			&r.ResponseStatus, &r.CreatedAt, &r.BusinessName)
		responses = append(responses, r)
	}

	c.JSON(http.StatusOK, gin.H{"responses": responses})
}

func sqlNullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
