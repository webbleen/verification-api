package database

import (
	"time"
	"verification-api/internal/models"

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
func GetActiveSubscription(projectID, userID string) (*models.Subscription, error) {
	var subscription models.Subscription
	err := DB.Where("project_id = ? AND user_id = ? AND status = ? AND expires_date > ?", 
		projectID, userID, "active", time.Now()).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetUserSubscriptions 获取用户的所有订阅（按项目）
func GetUserSubscriptions(projectID, userID string) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := DB.Where("project_id = ? AND user_id = ?", projectID, userID).Find(&subscriptions).Error
	return subscriptions, err
}

// CheckUserHasActiveSubscription 检查用户是否有有效订阅
func CheckUserHasActiveSubscription(projectID, userID string) (bool, error) {
	var count int64
	err := DB.Model(&models.Subscription{}).
		Where("project_id = ? AND user_id = ? AND status = ? AND expires_date > ?", 
			projectID, userID, "active", time.Now()).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetLatestSubscriptionByUser 获取用户的最新订阅（用于恢复购买）
func GetLatestSubscriptionByUser(projectID, userID string) (*models.Subscription, error) {
	var subscription models.Subscription
	err := DB.Where("project_id = ? AND user_id = ?", projectID, userID).
		Order("created_at DESC").
		First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// CreateOrUpdateSubscription 创建或更新订阅（按项目）
func CreateOrUpdateSubscription(subscription *models.Subscription) error {
	// 查找现有订阅（按项目、用户和原始交易ID）
	var existingSubscription models.Subscription
	err := DB.Where("project_id = ? AND user_id = ? AND original_transaction_id = ?", 
		subscription.ProjectID, subscription.UserID, subscription.OriginalTransactionID).First(&existingSubscription).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新订阅
			return DB.Create(subscription).Error
		}
		return err
	}

	// 更新现有订阅
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

	return DB.Save(&existingSubscription).Error
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
func GetAllUserSubscriptions(userID string) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&subscriptions).Error
	return subscriptions, err
}

