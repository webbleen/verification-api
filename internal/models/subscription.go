package models

import (
	"time"
)

// Subscription 订阅模型
// 存储用户的订阅信息，作为统一的订阅状态源
type Subscription struct {
	BaseModel

	// 关联字段
	AppAccountToken string `json:"app_account_token" gorm:"not null;index;column:app_account_token"` // App Account Token (UUID 格式)
	ProjectID       string `json:"project_id" gorm:"not null;index"`                                 // 项目ID，关联到project表
	Platform        string `json:"platform" gorm:"size:20;default:'ios';index"`                      // 平台：ios 或 android

	// 订阅状态字段
	Status string `json:"status" gorm:"not null;size:20;index"` // 订阅状态：active(激活)、inactive(未激活)、cancelled(已取消)、expired(过期)

	// 订阅时间字段
	StartDate time.Time `json:"start_date"` // 订阅开始时间
	EndDate   time.Time `json:"end_date"`   // 订阅结束时间

	// App Store / Google Play 相关字段
	ProductID             string    `json:"product_id" gorm:"size:100"`                    // 产品ID
	TransactionID         string    `json:"transaction_id" gorm:"size:100;uniqueIndex"`    // 交易ID
	OriginalTransactionID string    `json:"original_transaction_id" gorm:"size:100;index"` // 原始交易ID
	Environment           string    `json:"environment" gorm:"size:20"`                    // 环境：sandbox, production
	PurchaseDate          time.Time `json:"purchase_date"`                                 // 购买日期
	ExpiresDate           time.Time `json:"expires_date" gorm:"index"`                     // 过期日期
	AutoRenewStatus       bool      `json:"auto_renew_status"`                             // 自动续费状态

	// 收据相关字段（用于恢复购买）
	LatestReceipt     string `json:"latest_receipt" gorm:"type:text"`      // 最新收据（iOS base64 或 Android token）
	LatestReceiptInfo string `json:"latest_receipt_info" gorm:"type:text"` // 完整收据信息（JSON格式）
}
