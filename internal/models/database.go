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
	FromName     string `json:"from_name" gorm:"not null"`
	TemplateID   string `json:"template_id"`
	CustomConfig string `json:"custom_config" gorm:"type:text"` // JSON string
	IsActive     bool   `json:"is_active" gorm:"default:true"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	MaxRequests  int    `json:"max_requests" gorm:"default:1000"` // max requests per day

	// App 识别字段（用于订阅中心）
	BundleID    string `json:"bundle_id" gorm:"uniqueIndex"`    // iOS bundle ID，用于识别 iOS App
	PackageName string `json:"package_name" gorm:"uniqueIndex"` // Android package name，用于识别 Android App

	// Webhook 配置（用于通知 App Backend 订阅状态变化）
	WebhookCallbackURL string `json:"webhook_callback_url" gorm:"type:varchar(500)"` // App Backend 的 webhook 地址
	WebhookSecret      string `json:"webhook_secret" gorm:"type:varchar(255)"`       // 用于签名验证（可选）
}

// VerificationCode and RateLimit removed - using Redis only
