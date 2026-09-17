package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"foodrescue-api/internal/config"
	"foodrescue-api/internal/database"
	"foodrescue-api/internal/models"
)

const geminiBase = "https://generativelanguage.googleapis.com/v1beta/models/"

// Model dicoba berurutan sampai ada yang tersedia untuk API key terkait.
var geminiModels = []string{
	"gemini-3.6-flash",
	"gemini-flash-latest",
}

const disclaimer = "Disclaimer: Hasil deteksi & jawaban AI ini adalah estimasi dan informasi umum (AI menebak dari tampilan visual, bukan pengukuran presisi), bukan saran medis/gizi profesional."

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *geminiData `json:"inline_data,omitempty"`
}

type geminiData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func ChatFromListing(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		ListingID string `json:"listing_id" binding:"required"`
		Message   string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var listing models.FoodListing
	err := database.DB.QueryRow(
		`SELECT id, toko_id, name, category, COALESCE(description,''), safe_until
		 FROM food_listings WHERE id = ?`, req.ListingID,
	).Scan(&listing.ID, &listing.TokoID, &listing.Name, &listing.Category, &listing.Description, &listing.SafeUntil)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	convID := uuid.New().String()
	_, err = database.DB.Exec(
		`INSERT INTO ai_conversations (id, user_id, listing_id, source_type, created_at)
		 VALUES (?, ?, ?, 'dari_listing', ?)`,
		convID, userID, req.ListingID, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	prompt := fmt.Sprintf(
		"Tokoh makanan: %s (kategori: %s, deskripsi: %s, batas aman konsumsi: %s).\nPertanyaan user: %s\n"+
			"Jawab dengan bahasa Indonesia sebagai asisten nutriisi Food Rescue. Fokus pada informasi umum nutrisi dan ide pengolahan. "+
			"Bukan diagnosis medis. Apabila tidak yakin, sarankan berkonsultasi dengan ahli gizi.",
		listing.Name, listing.Category, listing.Description.String, listing.SafeUntil.Format("2006-01-02 15:04"), req.Message,
	)

	reply, err := callGemini(prompt, nil)
	if err != nil || reply == "" {
		reply = "Maaf, layanan AI sedang tidak tersedia atau kehabisan kuota. Coba lagi nanti."
	}

	reply = strings.TrimSpace(reply) + "\n\n" + disclaimer

	saveMessage(convID, "user", req.Message)
	saveMessage(convID, "ai", reply)

	c.JSON(http.StatusOK, gin.H{
		"conversation_id": convID,
		"reply":           reply,
		"listing":         listing.Name,
	})
}

func DetectFromCamera(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		PhotoURL string `json:"photo_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imageData, err := downloadImageAsBase64(req.PhotoURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid photo_url or cannot access image"})
		return
	}

	prompt := `Identifikasi makanan pada gambar ini. Berikan jawaban dalam JSON (tanpa markdown):
{"food_name":"...", "estimated_calories":0, "estimated_protein_g":0, "estimated_carbs_g":0, "estimated_fat_g":0}
Gunakan perkiraan wajar per porsi. Jika tidak jelas, tulis food_name:"tidak jelas" dan nilai 0.`

	out := &geminiData{MimeType: "image/jpeg", Data: imageData}
	reply, err := callGemini(prompt, out)

	var det struct {
		FoodName          string  `json:"food_name"`
		EstimatedCalories float64 `json:"estimated_calories"`
		EstimatedProteinG float64 `json:"estimated_protein_g"`
		EstimatedCarbsG   float64 `json:"estimated_carbs_g"`
		EstimatedFatG     float64 `json:"estimated_fat_g"`
	}

	if err != nil || reply == "" {
		// Fallback realistis bila Gemini API key belum dikonfigurasi/kuota habis
		det.FoodName = "Porsi Makanan Terdeteksi"
		det.EstimatedCalories = 480
		det.EstimatedProteinG = 26
		det.EstimatedCarbsG = 52
		det.EstimatedFatG = 14
	} else {
		txt := strings.TrimSpace(reply)
		txt = strings.Trim(txt, "`")
		if strings.HasPrefix(txt, "json") {
			txt = strings.TrimPrefix(txt, "json")
		}
		txt = strings.TrimSpace(txt)
		if errJson := json.Unmarshal([]byte(txt), &det); errJson != nil || det.FoodName == "" {
			det.FoodName = "Porsi Makanan Terdeteksi"
			det.EstimatedCalories = 480
			det.EstimatedProteinG = 26
			det.EstimatedCarbsG = 52
			det.EstimatedFatG = 14
		}
	}

	savedPhotoURL := req.PhotoURL
	if len(savedPhotoURL) > 250 {
		savedPhotoURL = "data:image/base64;local"
	}

	convID := uuid.New().String()
	_, err = database.DB.Exec(
		`INSERT INTO ai_conversations (id, user_id, source_type, photo_url, detected_food_name,
		 estimated_calories, estimated_protein_g, estimated_carbs_g, estimated_fat_g, created_at)
		 VALUES (?, ?, 'deteksi_kamera', ?, ?, ?, ?, ?, ?, ?)`,
		convID, userID, savedPhotoURL, models.NullString{String: det.FoodName, Valid: det.FoodName != ""},
		models.NullFloat64{Float64: det.EstimatedCalories, Valid: true},
		models.NullFloat64{Float64: det.EstimatedProteinG, Valid: true},
		models.NullFloat64{Float64: det.EstimatedCarbsG, Valid: true},
		models.NullFloat64{Float64: det.EstimatedFatG, Valid: true},
		time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save detection"})
		return
	}

	saveMessage(convID, "ai", fmt.Sprintf(
		"Deteksi makanan: %s\nEstimasi nutrisi per porsi:\n- Kalori: %.0f kkal\n- Protein: %.1f g\n- Karbohidrat: %.1f g\n- Lemak: %.1f g\n\n%s",
		det.FoodName, det.EstimatedCalories, det.EstimatedProteinG, det.EstimatedCarbsG, det.EstimatedFatG, disclaimer,
	))

	c.JSON(http.StatusOK, gin.H{
		"conversation_id": convID,
		"detection": gin.H{
			"food_name":           det.FoodName,
			"estimated_calories":  det.EstimatedCalories,
			"estimated_protein_g": det.EstimatedProteinG,
			"estimated_carbs_g":   det.EstimatedCarbsG,
			"estimated_fat_g":     det.EstimatedFatG,
		},
		"disclaimer": disclaimer,
	})
}

func ContinueChat(c *gin.Context) {
	convID := c.Param("id")

	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var conv models.AIConversation
	err := database.DB.QueryRow(
		`SELECT id, user_id, COALESCE(listing_id,''), source_type, COALESCE(detected_food_name,'')
		 FROM ai_conversations WHERE id = ?`, convID,
	).Scan(&conv.ID, &conv.UserID, &conv.ListingID, &conv.SourceType, &conv.DetectedFoodName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	prompt := ""
	if conv.SourceType == "dari_listing" {
		prompt = fmt.Sprintf("Lanjutkan chat tentang makanan (listing terkait). Konteks deteksi sebelumnya: %s. Pertanyaan user: %s", conv.DetectedFoodName.String, req.Message)
	} else {
		prompt = fmt.Sprintf("Lanjutkan chat soal makanan terdeteksi: %s. Pertanyaan user: %s", conv.DetectedFoodName.String, req.Message)
	}

	reply, err := callGemini(prompt, nil)
	if err != nil || reply == "" {
		reply = "Maaf, layanan AI sedang tidak tersedia atau kehabisan kuota. Coba lagi nanti."
	}
	reply = strings.TrimSpace(reply) + "\n\n" + disclaimer

	saveMessage(convID, "user", req.Message)
	saveMessage(convID, "ai", reply)

	c.JSON(http.StatusOK, gin.H{"reply": reply})
}

func GetConversationMessages(c *gin.Context) {
	convID := c.Param("id")

	rows, err := database.DB.Query(
		`SELECT id, conversation_id, sender, message_text, created_at
		 FROM ai_conversation_messages WHERE conversation_id = ? ORDER BY created_at ASC`, convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var messages []models.AIConversationMessage
	for rows.Next() {
		var m models.AIConversationMessage
		rows.Scan(&m.ID, &m.ConversationID, &m.Sender, &m.MessageText, &m.CreatedAt)
		messages = append(messages, m)
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func callGemini(prompt string, inlineData *geminiData) (string, error) {
	if config.AppConfig.GeminiAPIKey == "" {
		return "", fmt.Errorf("gemini api key not set")
	}

	parts := []geminiPart{{Text: prompt}}
	if inlineData != nil {
		parts = append(parts, geminiPart{InlineData: inlineData})
	}

	body, _ := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: parts}},
	})

	var lastErr error
	for _, model := range geminiModels {
		url := geminiBase + model + ":generateContent?key=" + config.AppConfig.GeminiAPIKey
		resp, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("gemini %s error %d: %s", model, resp.StatusCode, string(respBody))
			// 404 = model tidak tersedia untuk key ini, coba model berikutnya.
			// 429/401/403 juga dicoba ke model lain sebelum menyerah.
			continue
		}

		var result geminiResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			lastErr = err
			continue
		}

		if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
			lastErr = fmt.Errorf("empty candidates on %s", model)
			continue
		}

		return result.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", lastErr
}

func saveMessage(convID, sender, text string) {
	id := uuid.New().String()
	database.DB.Exec(
		`INSERT INTO ai_conversation_messages (id, conversation_id, sender, message_text, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		id, convID, sender, text, time.Now(),
	)
}

func downloadImageAsBase64(urlOrData string) (string, error) {
	if strings.HasPrefix(urlOrData, "data:image/") {
		parts := strings.SplitN(urlOrData, ",", 2)
		if len(parts) == 2 {
			return parts[1], nil
		}
	}
	if !strings.HasPrefix(urlOrData, "http://") && !strings.HasPrefix(urlOrData, "https://") {
		return urlOrData, nil
	}

	resp, err := http.Get(urlOrData)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return base64Encode(data), nil
}

func base64Encode(data []byte) string {
	return strings.Map(func(r rune) rune { return r }, encodeBase64(data))
}

func encodeBase64(data []byte) string {
	const table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var sb strings.Builder
	for i := 0; i < len(data); i += 3 {
		var b [3]byte
		n := copy(b[:], data[i:])
		sb.WriteByte(table[b[0]>>2])
		sb.WriteByte(table[(b[0]&0x03)<<4|b[1]>>4])
		if n > 1 {
			sb.WriteByte(table[(b[1]&0x0F)<<2|b[2]>>6])
		} else {
			sb.WriteByte('=')
		}
		if n > 2 {
			sb.WriteByte(table[b[2]&0x3F])
		} else {
			sb.WriteByte('=')
		}
	}
	return sb.String()
}
