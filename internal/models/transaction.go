package models

import (
	"time"
)

// Transaction 通用交易表
// 存储所有 IAP 交易记录（订阅和一次性内购）
type Transaction struct {
	BaseModel

	// 关联字段
	ProjectID       string `json:"project_id" gorm:"not null;index"`       // 项目ID
	AppAccountToken string `json:"app_account_token" gorm:"size:36;index"` // App Account Token (UUID)

	// 交易标识
	TransactionID         string `json:"transaction_id" gorm:"not null;size:100;uniqueIndex"` // 交易ID
	OriginalTransactionID string `json:"original_transaction_id" gorm:"size:100;index"`       // 原始交易ID（用于关联续订）

	// 产品信息
	ProductID string `json:"product_id" gorm:"size:100"` // 产品ID

	// 交易类型
	Type string `json:"type" gorm:"not null;size:20;index"` // subscription（订阅）或 non_consumable（一次性内购）

	// 环境
	Environment string `json:"environment" gorm:"size:20"` // sandbox 或 production

	// 时间
	PurchasedAt time.Time `json:"purchased_at"` // 购买时间
}

// TableName 指定表名
func (Transaction) TableName() string {
	return "transactions"
}
