package report

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

func CreateReport(c *gin.Context) {
	userID := c.GetString("user_id")

	var req models.CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New().String()
	_, err := database.DB.Exec(
		`INSERT INTO reports (id, reporter_user_id, reported_entity_type, reported_entity_id, reason, report_status, created_at)
		 VALUES (?, ?, ?, ?, ?, 'menunggu', ?)`,
		id, userID, req.ReportedEntityType, req.ReportedEntityID, req.Reason, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create report"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"report_id": id, "message": "Report submitted"})
}

func GetMyReports(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := database.DB.Query(
		`SELECT id, reporter_user_id, reported_entity_type, reported_entity_id, reason, report_status, created_at
		 FROM reports WHERE reporter_user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var reports []models.Report
	for rows.Next() {
		var r models.Report
		rows.Scan(&r.ID, &r.ReporterUserID, &r.ReportedEntityType, &r.ReportedEntityID,
			&r.Reason, &r.ReportStatus, &r.CreatedAt)
		reports = append(reports, r)
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}
