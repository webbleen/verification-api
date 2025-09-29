package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel provides common fields for all database models
type BaseModel struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// Project represents a project configuration in database
type Project struct {
	BaseModel
	ProjectID    string `json:"project_id" gorm:"uniqueIndex;not null"`
	ProjectName  string `json:"project_name" gorm:"not null"`
	APIKey       string `json:"api_key" gorm:"uniqueIndex;not null"`
	FromEmail    string `json:"from_email" gorm:"not null"`
	FromName     string `json:"from_name" gorm:"not null"`
	TemplateID   string `json:"template_id"`
	CustomConfig string `json:"custom_config" gorm:"type:text"` // JSON string
	IsActive     bool   `json:"is_active" gorm:"default:true"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	WebhookURL   string `json:"webhook_url"`
	RateLimit    int    `json:"rate_limit" gorm:"default:60"`     // requests per hour
	MaxRequests  int    `json:"max_requests" gorm:"default:1000"` // max requests per day
}

// VerificationCode represents a verification code record
type VerificationCode struct {
	BaseModel
	ProjectID string     `json:"project_id" gorm:"not null;index"`
	Email     string     `json:"email" gorm:"not null;index"`
	Code      string     `json:"code" gorm:"not null"`
	IsUsed    bool       `json:"is_used" gorm:"default:false"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null;index"`
	UsedAt    *time.Time `json:"used_at"`
	IPAddress string     `json:"ip_address"`
	UserAgent string     `json:"user_agent"`
}

// VerificationLog represents verification attempt logs
type VerificationLog struct {
	BaseModel
	ProjectID   string    `json:"project_id" gorm:"not null;index"`
	Email       string    `json:"email" gorm:"not null;index"`
	Action      string    `json:"action" gorm:"not null"` // "send", "verify", "expire"
	Success     bool      `json:"success" gorm:"not null"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	ErrorMsg    string    `json:"error_msg"`
	RequestTime time.Time `json:"request_time" gorm:"not null"`
}

// RateLimit represents rate limiting records
type RateLimit struct {
	BaseModel
	ProjectID   string    `json:"project_id" gorm:"not null;index"`
	Email       string    `json:"email" gorm:"not null;index"`
	IPAddress   string    `json:"ip_address" gorm:"not null;index"`
	Count       int       `json:"count" gorm:"not null;default:1"`
	WindowStart time.Time `json:"window_start" gorm:"not null;index"`
	ExpiresAt   time.Time `json:"expires_at" gorm:"not null;index"`
}
