package main

import (
	"strconv"
	"time"
)

type Client struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	ClientID     string     `json:"client_id"`
	ClientSecret string     `json:"client_secret"`
	Usertoken    string     `json:"usertoken"`
	Passtoken    string     `json:"passtoken"`
	AuthToken    string     `json:"auth_token"`
	RefreshToken string     `json:"refresh_token"`
	BaseURL      string     `json:"base_url"`
	IsActive     bool       `json:"is_active"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type MasterViolation struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Raw  []byte `json:"raw,omitempty"`
}

type Camera struct {
	ID           int64      `json:"id"`
	ClientID     int64      `json:"client_id"`
	ClientName   string     `json:"client_name,omitempty"`
	CameraCode   string     `json:"camera_code"`
	DeviceName   string     `json:"device_name"`
	LocationName string     `json:"location_name"`
	Address      string     `json:"address"`
	Latitude     float64    `json:"latitude"`
	Longitude    float64    `json:"longitude"`
	Status       string     `json:"status"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type Violation struct {
	ID              int64     `json:"id"`
	ClientID        int64     `json:"client_id"`
	CameraID        int64     `json:"camera_id"`
	DeviceName      string    `json:"device_name"`
	Plate           string    `json:"plate"`
	PlateColor      string    `json:"plate_color"`
	PlateImageURL   string    `json:"plate_image_url"`
	VehicleType     string    `json:"vehicle_type"`
	VehicleColor    string    `json:"vehicle_color"`
	VehicleImageURL string    `json:"vehicle_image_url"`
	VideoURL        string    `json:"video_url"`
	ViolationCode   string    `json:"violation_code"`
	ViolationName   string    `json:"violation_name"`
	LocationName    string    `json:"location_name"`
	CaptureTime     int64     `json:"capture_time"`
	Status          string    `json:"status"`
	ResponseStatus  int       `json:"response_status"`
	Attempts        int       `json:"attempts"`
	ErrorMessage    string    `json:"error_message"`
	CreatedAt       time.Time `json:"created_at"`
	SentAt          *time.Time `json:"sent_at,omitempty"`
}

type ViolationItem struct {
	DeviceName      string `json:"deviceName"`
	Plate           string `json:"plate"`
	PlateColor      string `json:"plateColor"`
	PlateImageURL   string `json:"plateImageUrl"`
	VehicleType     string `json:"vehicleType"`
	VehicleColor    string `json:"vehicleColor"`
	VehicleImageURL string `json:"vehicleImageUrl"`
	VideoURL        string `json:"videoUrl"`
	ViolationCode   string `json:"violationCode"`
	ViolationName   string `json:"violationName"`
	LocationName    string `json:"locationName"`
	CaptureTime     int64  `json:"captureTime"`
}

type ViolationPayload struct {
	Datas []ViolationItem `json:"datas"`
}

type SendResponse struct {
	Status int `json:"status"`
}

// ViolationDetail = detail pelanggaran lengkap dengan nama client & kamera.
type ViolationDetail struct {
	Violation
	ClientName string `json:"client_name"`
	ClientCode string `json:"client_code"`
	CameraName string `json:"camera_name"`
	CameraCode string `json:"camera_code"`
}

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	IsActive     bool   `json:"is_active"`
}

type LoginResponse struct {
	Status       int    `json:"status"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refreshToken"`
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return "etle http " + strconv.Itoa(e.StatusCode) + ": " + e.Body
}

type SyncLog struct {
	ID            int64  `json:"id"`
	ClientID      *int64 `json:"client_id"`
	ClientName    string `json:"client_name,omitempty"`
	Action        string `json:"action"`
	Status        string `json:"status"`
	RecordsSynced int    `json:"records_synced"`
	Message       string `json:"message"`
	CreatedAt     string `json:"created_at"`
}

type DashboardStats struct {
	TotalViolations int64 `json:"total_violations"`
	Pending         int64 `json:"pending"`
	Processing      int64 `json:"processing"`
	Sent            int64 `json:"sent"`
	Failed          int64 `json:"failed"`
	TotalClients    int64 `json:"total_clients"`
	TotalCameras    int64 `json:"total_cameras"`
	MasterCount     int64 `json:"master_count"`
	RetentionDays   int   `json:"retention_days"`
}

