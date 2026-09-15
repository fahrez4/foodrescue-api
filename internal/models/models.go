package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ──────────────── NULL TYPES (JSON-friendly) ────────────────

// NullString mirip NullString tapi di-serialize JSON sebagai null
type NullString struct {
	String string
	Valid  bool
}

func (n NullString) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.String)
}

func (n *NullString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		n.String = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	n.Valid = true
	n.String = s
	return nil
}

func (n *NullString) Scan(value interface{}) error {
	if value == nil {
		n.String, n.Valid = "", false
		return nil
	}
	switch v := value.(type) {
	case []byte:
		n.String, n.Valid = string(v), true
	case string:
		n.String, n.Valid = v, true
	default:
		return errors.New("cannot scan type into NullString")
	}
	return nil
}

func (n NullString) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.String, nil
}

// NullFloat64 mirip NullFloat64 tapi di-serialize JSON sebagai null
type NullFloat64 struct {
	Float64 float64
	Valid   bool
}

func (n NullFloat64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Float64)
}

func (n *NullFloat64) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		n.Valid = false
		n.Float64 = 0
		return nil
	}
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	n.Valid = true
	n.Float64 = f
	return nil
}

func (n *NullFloat64) Scan(value interface{}) error {
	if value == nil {
		n.Float64, n.Valid = 0, false
		return nil
	}
	switch v := value.(type) {
	case float64:
		n.Float64, n.Valid = v, true
	case []byte:
		var f float64
		if _, err := fmt.Sscanf(string(v), "%f", &f); err != nil {
			return err
		}
		n.Float64, n.Valid = f, true
	default:
		return errors.New("cannot scan type into NullFloat64")
	}
	return nil
}

func (n NullFloat64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Float64, nil
}

// NullTime mirip NullTime tapi di-serialize JSON sebagai null
type NullTime struct {
	Time  time.Time
	Valid bool
}

func (n NullTime) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Time)
}

func (n *NullTime) Scan(value interface{}) error {
	if value == nil {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		n.Time, n.Valid = v, true
	case []byte:
		t, err := time.Parse("2006-01-02 15:04:05", string(v))
		if err == nil {
			n.Time, n.Valid = t, true
			return nil
		}
		t, err = time.Parse(time.RFC3339, string(v))
		if err == nil {
			n.Time, n.Valid = t, true
			return nil
		}
		return err
	default:
		return errors.New("cannot scan type into NullTime")
	}
	return nil
}

func (n NullTime) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Time, nil
}

// ──────────────── USERS ────────────────
type User struct {
	ID              string         `json:"id"`
	Email           string         `json:"email"`
	FullName        string         `json:"full_name"`
	PhotoURL        NullString `json:"photo_url"`
	PhoneNumber     NullString `json:"phone_number"`
	AuthProvider    string         `json:"auth_provider"`
	Role            string         `json:"role"`
	IsNGOVerified   bool           `json:"is_ngo_verified"`
	TrustScore      float64        `json:"trust_score"`
	Latitude        NullFloat64 `json:"latitude"`
	Longitude       NullFloat64 `json:"longitude"`
	AddressText     NullString `json:"address_text"`
	AccountStatus   string         `json:"account_status"`
	PasswordHash    NullString `json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ──────────────── PAYMENT METHODS ────────────────
type PaymentMethod struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Provider         string    `json:"provider"`
	AccountReference string    `json:"account_reference"`
	IsDefault        bool      `json:"is_default"`
	CreatedAt        time.Time `json:"created_at"`
}

// ──────────────── TOKO PROFILES ────────────────
type TokoProfile struct {
	ID                 string         `json:"id"`
	UserID             string         `json:"user_id"`
	BusinessName       string         `json:"business_name"`
	BusinessCategory   string         `json:"business_category"`
	Address            string         `json:"address"`
	Latitude           NullFloat64 `json:"latitude"`
	Longitude          NullFloat64 `json:"longitude"`
	LegalDocumentURL   NullString `json:"legal_document_url"`
	OperationalHours   NullString `json:"operational_hours"`
	VerificationStatus string         `json:"verification_status"`
	VerifiedByAdminID  NullString `json:"verified_by_admin_id"`
	VerifiedAt         NullTime   `json:"verified_at"`
	AverageRating      float64        `json:"average_rating"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// ──────────────── COURIER PROFILES ────────────────
type CourierProfile struct {
	ID                  string          `json:"id"`
	UserID              string          `json:"user_id"`
	VehicleType         string          `json:"vehicle_type"`
	IDDocumentURL       NullString  `json:"id_document_url"`
	VerificationStatus  string          `json:"verification_status"`
	VerifiedByAdminID   NullString  `json:"verified_by_admin_id"`
	VerifiedAt          NullTime    `json:"verified_at"`
	IsOnline            bool            `json:"is_online"`
	CurrentLatitude     NullFloat64 `json:"current_latitude"`
	CurrentLongitude    NullFloat64 `json:"current_longitude"`
	LastLocationUpdate  NullTime    `json:"last_location_update"`
	AverageRating       float64         `json:"average_rating"`
	TotalEarnings       float64         `json:"total_earnings"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// ──────────────── FOOD LISTINGS ────────────────
type FoodListing struct {
	ID              string         `json:"id"`
	TokoID          string         `json:"toko_id"`
	Name            string         `json:"name"`
	Category        string         `json:"category"`
	Description     NullString `json:"description"`
	PhotoURL        NullString `json:"photo_url"`
	InitialPrice    float64        `json:"initial_price"`
	MinimumPrice    float64        `json:"minimum_price"`
	CurrentPrice    float64        `json:"current_price"`
	StockQuantity   int            `json:"stock_quantity"`
	FoodSafetyNotes NullString `json:"food_safety_notes"`
	SafeUntil       time.Time      `json:"safe_until"`
	PickupStartTime NullTime   `json:"pickup_start_time"`
	PickupEndTime   NullTime   `json:"pickup_end_time"`
	Status          string         `json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ──────────────── ORDERS ────────────────
type Order struct {
	ID                string        `json:"id"`
	ListingID         string        `json:"listing_id"`
	UserID            string        `json:"user_id"`
	Quantity          int           `json:"quantity"`
	PriceAtPurchase   float64       `json:"price_at_purchase"`
	TotalAmount       float64       `json:"total_amount"`
	FulfillmentMethod string        `json:"fulfillment_method"`
	PaymentMethod     string        `json:"payment_method"`
	PaymentStatus     string        `json:"payment_status"`
	OrderStatus       string        `json:"order_status"`
	ConfirmationCode  string        `json:"confirmation_code"`
	CreatedAt         time.Time     `json:"created_at"`
	CompletedAt       NullTime  `json:"completed_at"`
}

// ──────────────── DELIVERIES ────────────────
type Delivery struct {
	ID                       string         `json:"id"`
	OrderID                  string         `json:"order_id"`
	CourierID                NullString `json:"courier_id"`
	MatchingStatus           string         `json:"matching_status"`
	PickupLatitude           NullFloat64 `json:"pickup_latitude"`
	PickupLongitude          NullFloat64 `json:"pickup_longitude"`
	DropoffLatitude          NullFloat64 `json:"dropoff_latitude"`
	DropoffLongitude         NullFloat64 `json:"dropoff_longitude"`
	TripStatus               NullString `json:"trip_status"`
	PickupConfirmationCode   NullString `json:"pickup_confirmation_code"`
	DropoffConfirmationCode  NullString `json:"dropoff_confirmation_code"`
	OfferExpiresAt           NullTime   `json:"offer_expires_at"`
	DeliveryFee              float64        `json:"delivery_fee"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
}

// ──────────────── COMMUNITY POSTS ────────────────
type CommunityPost struct {
	ID              string         `json:"id"`
	ListingID       string         `json:"listing_id"`
	TokoID          string         `json:"toko_id"`
	TargetCategory  string         `json:"target_category"`
	TransportFee    float64        `json:"transport_fee"`
	ClaimStatus     string         `json:"claim_status"`
	ClaimedByUserID NullString `json:"claimed_by_user_id"`
	ClaimedAt       NullTime   `json:"claimed_at"`
	CompletedAt     NullTime   `json:"completed_at"`
	CreatedAt       time.Time      `json:"created_at"`
}

// ──────────────── EMERGENCY ALERTS ────────────────
type EmergencyAlert struct {
	ID           string    `json:"id"`
	NGOUserID    string    `json:"ngo_user_id"`
	NeedType     string    `json:"need_type"`
	TargetArea   string    `json:"target_area"`
	UrgencyLevel string    `json:"urgency_level"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type EmergencyAlertResponse struct {
	ID               string         `json:"id"`
	AlertID          string         `json:"alert_id"`
	TokoID           string         `json:"toko_id"`
	AvailabilityNote NullString `json:"availability_note"`
	ResponseStatus   string         `json:"response_status"`
	CreatedAt        time.Time      `json:"created_at"`
}

// ──────────────── IMPACT REPORTS ────────────────
type ImpactReport struct {
	ID                 string    `json:"id"`
	OwnerUserID        string    `json:"owner_user_id"`
	PeriodType         string    `json:"period_type"`
	PeriodStart        string    `json:"period_start"`
	PeriodEnd          string    `json:"period_end"`
	TotalFoodSavedKg   float64   `json:"total_food_saved_kg"`
	TotalMoneyAmount   float64   `json:"total_money_amount"`
	EstimatedCO2SavedKg float64 `json:"estimated_co2_saved_kg"`
	GeneratedAt        time.Time `json:"generated_at"`
}

// ──────────────── RECIPES ────────────────
type Recipe struct {
	ID              string    `json:"id"`
	FoodCategory    string    `json:"food_category"`
	Title           string    `json:"title"`
	IngredientsText string    `json:"ingredients_text"`
	StepsText       string    `json:"steps_text"`
	CreatedAt       time.Time `json:"created_at"`
}

// ──────────────── AI CONVERSATIONS ────────────────
type AIConversation struct {
	ID                string         `json:"id"`
	UserID            string         `json:"user_id"`
	ListingID         NullString `json:"listing_id"`
	SourceType        string         `json:"source_type"`
	PhotoURL          NullString `json:"photo_url"`
	DetectedFoodName  NullString `json:"detected_food_name"`
	EstimatedCalories NullFloat64 `json:"estimated_calories"`
	EstimatedProteinG NullFloat64 `json:"estimated_protein_g"`
	EstimatedCarbsG   NullFloat64 `json:"estimated_carbs_g"`
	EstimatedFatG     NullFloat64 `json:"estimated_fat_g"`
	CreatedAt         time.Time      `json:"created_at"`
}

type AIConversationMessage struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Sender         string    `json:"sender"`
	MessageText    string    `json:"message_text"`
	CreatedAt      time.Time `json:"created_at"`
}

// ──────────────── CHATS / MESSAGES ────────────────
type Chat struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID          string    `json:"id"`
	ChatID      string    `json:"chat_id"`
	SenderID    string    `json:"sender_id"`
	MessageText string    `json:"message_text"`
	IsRead      bool      `json:"is_read"`
	CreatedAt   time.Time `json:"created_at"`
}

// ──────────────── RATINGS ────────────────
type Rating struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	RaterUserID   string    `json:"rater_user_id"`
	RateeUserID   string    `json:"ratee_user_id"`
	RatingTarget  string    `json:"rating_target"`
	Score         int       `json:"score"`
	ReviewText    string    `json:"review_text"`
	CreatedAt     time.Time `json:"created_at"`
}

// ──────────────── REPORTS ────────────────
type Report struct {
	ID                string         `json:"id"`
	ReporterUserID    string         `json:"reporter_user_id"`
	ReportedEntityType string       `json:"reported_entity_type"`
	ReportedEntityID  string         `json:"reported_entity_id"`
	Reason            string         `json:"reason"`
	ReportStatus      string         `json:"report_status"`
	ReviewedByAdminID NullString `json:"reviewed_by_admin_id"`
	ReviewedAt        NullTime   `json:"reviewed_at"`
	CreatedAt         time.Time      `json:"created_at"`
}

// ──────────────── REQUEST / RESPONSE DTOs ────────────────

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"omitempty,min=6"`
	FullName    string `json:"full_name" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"omitempty"`
	Role        string `json:"role" binding:"required,oneof=user toko kurir"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type RegisterTokoRequest struct {
	BusinessName     string `json:"business_name" binding:"required"`
	BusinessCategory string `json:"business_category" binding:"required"`
	Address          string `json:"address" binding:"required"`
	PhoneNumber      string `json:"phone_number" binding:"required"`
	LegalDocumentURL string `json:"legal_document_url" binding:"omitempty"`
	OperationalHours string `json:"operational_hours" binding:"omitempty"`
}

type RegisterKurirRequest struct {
	PhoneNumber  string `json:"phone_number" binding:"required"`
	VehicleType  string `json:"vehicle_type" binding:"required"`
	IDDocumentURL string `json:"id_document_url" binding:"omitempty"`
}

type CreateListingRequest struct {
	Name            string  `json:"name" binding:"required"`
	Category        string  `json:"category" binding:"required"`
	Description     string  `json:"description" binding:"omitempty"`
	PhotoURL        string  `json:"photo_url" binding:"omitempty"`
	InitialPrice    float64 `json:"initial_price" binding:"required,gt=0"`
	MinimumPrice    float64 `json:"minimum_price" binding:"required,gt=0"`
	StockQuantity   int     `json:"stock_quantity" binding:"required,gte=1"`
	FoodSafetyNotes string  `json:"food_safety_notes" binding:"omitempty"`
	SafeUntil       string  `json:"safe_until" binding:"required"`
	PickupStartTime string  `json:"pickup_start_time" binding:"omitempty"`
	PickupEndTime   string  `json:"pickup_end_time" binding:"omitempty"`
}

type CreateOrderRequest struct {
	ListingID         string `json:"listing_id" binding:"required"`
	Quantity          int    `json:"quantity" binding:"required,gte=1"`
	FulfillmentMethod string `json:"fulfillment_method" binding:"required,oneof=pickup_mandiri diantar_kurir"`
	PaymentMethod     string `json:"payment_method" binding:"omitempty"`
}

type UpdateOrderStatusRequest struct {
	OrderStatus string `json:"order_status" binding:"required"`
}

type CreateRatingRequest struct {
	OrderID       string `json:"order_id" binding:"required"`
	RateeUserID   string `json:"ratee_user_id" binding:"required"`
	RatingTarget  string `json:"rating_target" binding:"required,oneof=toko kurir user"`
	Score         int    `json:"score" binding:"required,gte=1,lte=5"`
	ReviewText    string `json:"review_text" binding:"omitempty"`
}

type CreateReportRequest struct {
	ReportedEntityType string `json:"reported_entity_type" binding:"required,oneof=listing user toko kurir order community_post"`
	ReportedEntityID   string `json:"reported_entity_id" binding:"required"`
	Reason             string `json:"reason" binding:"required"`
}

type CreateChatRequest struct {
	OrderID string `json:"order_id" binding:"required"`
}

type SendMessageRequest struct {
	MessageText string `json:"message_text" binding:"required"`
}

type CreateCommunityPostRequest struct {
	ListingID      string  `json:"listing_id" binding:"required"`
	TargetCategory string  `json:"target_category" binding:"required,oneof=kompos pakan_ternak lainnya"`
	TransportFee   float64 `json:"transport_fee" binding:"gte=0"`
}

type ClaimCommunityPostRequest struct {
	CommunityPostID string `json:"community_post_id" binding:"required"`
}

type CreateEmergencyAlertRequest struct {
	NeedType     string `json:"need_type" binding:"required"`
	TargetArea   string `json:"target_area" binding:"required"`
	UrgencyLevel string `json:"urgency_level" binding:"required,oneof=rendah sedang tinggi kritis"`
	Description  string `json:"description" binding:"omitempty"`
}

type RespondEmergencyAlertRequest struct {
	AlertID          string `json:"alert_id" binding:"required"`
	AvailabilityNote string `json:"availability_note" binding:"omitempty"`
}

type AdminVerifyRequest struct {
	Status string `json:"status" binding:"required,oneof=approved rejected"`
}

type UpdateListingRequest struct {
	Name            string  `json:"name" binding:"omitempty"`
	Category        string  `json:"category" binding:"omitempty"`
	Description     string  `json:"description" binding:"omitempty"`
	PhotoURL        string  `json:"photo_url" binding:"omitempty"`
	InitialPrice    float64 `json:"initial_price" binding:"omitempty,gt=0"`
	MinimumPrice    float64 `json:"minimum_price" binding:"omitempty,gt=0"`
	StockQuantity   int     `json:"stock_quantity" binding:"omitempty,gte=0"`
	FoodSafetyNotes string  `json:"food_safety_notes" binding:"omitempty"`
	SafeUntil       string  `json:"safe_until" binding:"omitempty"`
	Status          string  `json:"status" binding:"omitempty,oneof=active inactive sold_out expired pushed_to_community discarded"`
}

type UpdateProfileRequest struct {
	FullName    string  `json:"full_name" binding:"omitempty"`
	PhoneNumber string  `json:"phone_number" binding:"omitempty"`
	PhotoURL    string  `json:"photo_url" binding:"omitempty"`
	AddressText string  `json:"address_text" binding:"omitempty"`
	Latitude    float64 `json:"latitude" binding:"omitempty"`
	Longitude   float64 `json:"longitude" binding:"omitempty"`
}

type UpdateCourierLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

type ToggleOnlineRequest struct {
	IsOnline bool `json:"is_online"`
}

type AcceptDeliveryRequest struct {
	DeliveryID string `json:"delivery_id" binding:"required"`
}

type UpdateTripStatusRequest struct {
	TripStatus string `json:"trip_status" binding:"required,oneof=menuju_toko barang_diambil menuju_user diterima_user"`
}

type ConfirmPickupRequest struct {
	ConfirmationCode string `json:"confirmation_code" binding:"required"`
}

type ConfirmDropoffRequest struct {
	ConfirmationCode string `json:"confirmation_code" binding:"required"`
}

type AIChatRequest struct {
	ListingID  string `json:"listing_id" binding:"omitempty"`
	Message    string `json:"message" binding:"required"`
	PhotoURL   string `json:"photo_url" binding:"omitempty"`
	Source     string `json:"source" binding:"required,oneof=dari_listing deteksi_kamera"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	Total      int         `json:"total"`
	TotalPages int         `json:"total_pages"`
}
