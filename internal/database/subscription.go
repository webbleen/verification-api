package database

import (
	"time"
	"verification-api/internal/models"
	"verification-api/pkg/logging"

	"gorm.io/gorm"
)

// CreateSubscription 创建订阅
func CreateSubscription(subscription *models.Subscription) error {
	return DB.Create(subscription).Error
}

// UpdateSubscription 更新订阅
func UpdateSubscription(subscription *models.Subscription) error {
	return DB.Save(subscription).Error
}

// GetSubscriptionByTransactionID 通过交易ID获取订阅（按项目）
func GetSubscriptionByTransactionID(projectID, transactionID string) (*models.Subscription, error) {
	var subscription models.Subscription
	err := DB.Where("project_id = ? AND transaction_id = ?", projectID, transactionID).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetSubscriptionByOriginalTransactionID 通过原始交易ID获取订阅（按项目）
func GetSubscriptionByOriginalTransactionID(projectID, originalTransactionID string) (*models.Subscription, error) {
	var subscription models.Subscription
	err := DB.Where("project_id = ? AND original_transaction_id = ?", projectID, originalTransactionID).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetActiveSubscription 获取用户的活跃订阅（按项目）
func GetActiveSubscription(projectID, appAccountToken string) (*models.Subscription, error) {
	var subscription models.Subscription
	err := DB.Where("project_id = ? AND app_account_token = ? AND status = ? AND expires_date > ?",
		projectID, appAccountToken, "active", time.Now()).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetUserSubscriptions 获取用户的所有订阅（按项目）
func GetUserSubscriptions(projectID, appAccountToken string) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := DB.Where("project_id = ? AND app_account_token = ?", projectID, appAccountToken).Find(&subscriptions).Error
	return subscriptions, err
}

// CheckUserHasActiveSubscription 检查用户是否有有效订阅
func CheckUserHasActiveSubscription(projectID, appAccountToken string) (bool, error) {
	var count int64
	err := DB.Model(&models.Subscription{}).
		Where("project_id = ? AND app_account_token = ? AND status = ? AND expires_date > ?",
			projectID, appAccountToken, "active", time.Now()).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetLatestSubscriptionByUser 获取用户的最新订阅（用于恢复购买）
func GetLatestSubscriptionByUser(projectID, appAccountToken string) (*models.Subscription, error) {
	var subscription models.Subscription
	err := DB.Where("project_id = ? AND app_account_token = ?", projectID, appAccountToken).
		Order("created_at DESC").
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// CreateOrUpdateSubscription 创建或更新订阅（按项目）
// 优先通过 original_transaction_id 查找，支持绑定 user_id
// 使用数据库事务确保并发安全
func CreateOrUpdateSubscription(subscription *models.Subscription) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// 首先通过 project_id + original_transaction_id 查找（不考虑 uuid）
		// 这样可以找到 webhook 创建的 uuid 为空的订阅
		// 使用 SELECT FOR UPDATE 锁定行，防止并发问题
		var existingSubscription models.Subscription
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("project_id = ? AND original_transaction_id = ?",
				subscription.ProjectID, subscription.OriginalTransactionID).
			First(&existingSubscription).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 创建新订阅
				return tx.Create(subscription).Error
			}
			return err
		}

		// 更新现有订阅
		// 处理 appAccountToken 绑定逻辑
		if existingSubscription.AppAccountToken == "" {
			// 如果现有订阅的 appAccountToken 为空，且新订阅有 appAccountToken，则绑定
			if subscription.AppAccountToken != "" {
				logging.Infof("Binding appAccountToken to subscription - original_transaction_id: %s, app_account_token: %s",
					subscription.OriginalTransactionID, subscription.AppAccountToken)
				existingSubscription.AppAccountToken = subscription.AppAccountToken
			}
		} else {
			// 如果现有订阅已有 appAccountToken
			if subscription.AppAccountToken != "" && existingSubscription.AppAccountToken != subscription.AppAccountToken {
				// appAccountToken 不匹配，这可能表示：
				// 1. 同一个 original_transaction_id 被多个用户使用（不应该发生，因为 original_transaction_id 是唯一的）
				// 2. 数据不一致或并发冲突
				// 为了安全，我们保留原有的 appAccountToken，不覆盖
				logging.Errorf("AppAccountToken mismatch detected - original_transaction_id: %s, existing_app_account_token: %s, new_app_account_token: %s. Keeping existing app_account_token.",
					subscription.OriginalTransactionID, existingSubscription.AppAccountToken, subscription.AppAccountToken)
			} else if existingSubscription.AppAccountToken == subscription.AppAccountToken {
				// appAccountToken 匹配，正常更新
				logging.Infof("Updating subscription - original_transaction_id: %s, app_account_token: %s",
					subscription.OriginalTransactionID, subscription.AppAccountToken)
			}
			// 注意：这里不更新 appAccountToken，保持原有值
		}

		// 更新其他字段
		existingSubscription.Status = subscription.Status
		existingSubscription.StartDate = subscription.StartDate
		existingSubscription.EndDate = subscription.EndDate
		existingSubscription.ExpiresDate = subscription.ExpiresDate
		existingSubscription.AutoRenewStatus = subscription.AutoRenewStatus
		existingSubscription.LatestReceipt = subscription.LatestReceipt
		existingSubscription.LatestReceiptInfo = subscription.LatestReceiptInfo
		existingSubscription.Plan = subscription.Plan
		existingSubscription.ProductID = subscription.ProductID
		existingSubscription.TransactionID = subscription.TransactionID
		existingSubscription.Environment = subscription.Environment
		existingSubscription.PurchaseDate = subscription.PurchaseDate

		return tx.Save(&existingSubscription).Error
	})
}

// FindSubscriptionByOriginalTransactionID finds subscription by original transaction ID (across all projects)
func FindSubscriptionByOriginalTransactionID(originalTransactionID string) (*models.Subscription, error) {
	var subscription models.Subscription
	err := DB.Where("original_transaction_id = ?", originalTransactionID).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// FindSubscriptionByPurchaseToken finds subscription by purchase token (Android)
func FindSubscriptionByPurchaseToken(purchaseToken string) (*models.Subscription, error) {
	var subscription models.Subscription
	// Purchase token is stored in LatestReceipt field for Android
	err := DB.Where("platform = ? AND latest_receipt = ?", "android", purchaseToken).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetAllUserSubscriptions gets all subscriptions for a user across all projects
func GetAllUserSubscriptions(appAccountToken string) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := DB.Where("app_account_token = ?", appAccountToken).Order("created_at DESC").Find(&subscriptions).Error
	return subscriptions, err
}
