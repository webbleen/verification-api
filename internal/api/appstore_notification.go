package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/internal/services"
	"verification-api/pkg/logging"

	"github.com/gin-gonic/gin"
)

// processAppStoreNotification processes App Store notification
func processAppStoreNotification(environment string, c *gin.Context) {
	startTime := time.Now()

	// Read raw body
	body, err := c.GetRawData()
	if err != nil {
		logging.Errorf("Failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to read request body",
		})
		return
	}

	// Parse notification
	var notification models.AppStoreNotification
	if err := json.Unmarshal(body, &notification); err != nil {
		logging.Errorf("Failed to parse notification: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid notification format",
		})
		return
	}

	// Handle heartbeat
	if notification.NotificationType == "" {
		logging.Infof("AppStore heartbeat - environment: %s", environment)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"status":  "heartbeat_ok",
		})
		return
	}

	// Get project by bundle_id
	projectService := services.NewProjectService()
	project, err := projectService.GetProjectByBundleID(notification.Data.BundleID)
	if err != nil {
		logging.Errorf("Project not found for bundle_id: %s, error: %v", notification.Data.BundleID, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project not found for bundle_id: " + notification.Data.BundleID,
		})
		return
	}

	// Parse transaction info
	transactionInfo, err := parseTransactionInfo(notification.Data.SignedTransactionInfo)
	if err != nil {
		logging.Errorf("Failed to parse transaction info: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to parse transaction info",
		})
		return
	}

	// Handle notification by type
	if err := handleNotificationByType(notification.NotificationType, transactionInfo, project.ProjectID, notification.Data.Environment); err != nil {
		logging.Errorf("Failed to handle notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to process notification",
		})
		return
	}

	processingTime := time.Since(startTime)
	logging.Infof("AppStore notification processed - type: %s, transaction: %s, time: %v",
		notification.NotificationType, transactionInfo.TransactionID, processingTime)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification processed successfully",
	})
}

// AppStoreProductionNotificationHandler handles production environment notifications
// POST /api/appstore/notifications/production
func AppStoreProductionNotificationHandler(c *gin.Context) {
	processAppStoreNotification("production", c)
}

// AppStoreSandboxNotificationHandler handles sandbox environment notifications
// POST /api/appstore/notifications/sandbox
func AppStoreSandboxNotificationHandler(c *gin.Context) {
	processAppStoreNotification("sandbox", c)
}

// parseTransactionInfo parses transaction info from base64 string
func parseTransactionInfo(signedTransactionInfo string) (*models.TransactionInfo, error) {
	decoded, err := base64.StdEncoding.DecodeString(signedTransactionInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction info: %w", err)
	}

	var transactionInfo models.TransactionInfo
	if err := json.Unmarshal(decoded, &transactionInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction info: %w", err)
	}

	return &transactionInfo, nil
}

// handleNotificationByType handles notification by type
func handleNotificationByType(notificationType string, transactionInfo *models.TransactionInfo, projectID, environment string) error {
	switch notificationType {
	case "INITIAL_BUY", "SUBSCRIBED":
		return handleInitialBuy(transactionInfo, projectID, environment)
	case "DID_RENEW", "RENEWAL_EXTENDED":
		return handleDidRenew(transactionInfo, projectID)
	case "DID_FAIL_TO_RENEW":
		return handleDidFailToRenew(transactionInfo, projectID)
	case "DID_CANCEL":
		return handleDidCancel(transactionInfo, projectID)
	case "DID_REFUND", "REVOKE":
		return handleDidRefund(transactionInfo, projectID)
	case "EXPIRED", "GRACE_PERIOD_EXPIRED":
		return handleExpired(transactionInfo, projectID)
	default:
		logging.Infof("Unknown notification type: %s", notificationType)
		return nil
	}
}

// handleInitialBuy handles initial purchase
func handleInitialBuy(transactionInfo *models.TransactionInfo, projectID, environment string) error {
	logging.Infof("Handling INITIAL_BUY - transaction: %s", transactionInfo.TransactionID)

	// Find existing subscription by original transaction ID
	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		// Create new subscription
		subscription = &models.Subscription{
			ProjectID:             projectID,
			UserID:                "", // Will be set when user verifies receipt
			Platform:              "ios",
			Plan:                  getPlanFromProductID(transactionInfo.ProductID),
			Status:                "active",
			StartDate:             time.Unix(transactionInfo.PurchaseDateMS/1000, 0),
			EndDate:               time.Unix(transactionInfo.ExpiresDateMS/1000, 0),
			ProductID:             transactionInfo.ProductID,
			TransactionID:         transactionInfo.TransactionID,
			OriginalTransactionID: transactionInfo.OriginalTransactionID,
			Environment:           environment,
			PurchaseDate:          time.Unix(transactionInfo.PurchaseDateMS/1000, 0),
			ExpiresDate:           time.Unix(transactionInfo.ExpiresDateMS/1000, 0),
			AutoRenewStatus:       transactionInfo.AutoRenewStatus == 1,
		}
		return database.CreateSubscription(subscription)
	}

	// Update existing subscription
	subscription.Status = "active"
	subscription.ExpiresDate = time.Unix(transactionInfo.ExpiresDateMS/1000, 0)
	subscription.AutoRenewStatus = transactionInfo.AutoRenewStatus == 1
	return database.UpdateSubscription(subscription)
}

// handleDidRenew handles renewal
func handleDidRenew(transactionInfo *models.TransactionInfo, projectID string) error {
	logging.Infof("Handling DID_RENEW - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	subscription.Status = "active"
	subscription.ExpiresDate = time.Unix(transactionInfo.ExpiresDateMS/1000, 0)
	subscription.AutoRenewStatus = transactionInfo.AutoRenewStatus == 1
	return database.UpdateSubscription(subscription)
}

// handleDidFailToRenew handles failed renewal
func handleDidFailToRenew(transactionInfo *models.TransactionInfo, projectID string) error {
	logging.Infof("Handling DID_FAIL_TO_RENEW - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	subscription.Status = "failed"
	subscription.AutoRenewStatus = false
	return database.UpdateSubscription(subscription)
}

// handleDidCancel handles cancellation
func handleDidCancel(transactionInfo *models.TransactionInfo, projectID string) error {
	logging.Infof("Handling DID_CANCEL - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	subscription.Status = "cancelled"
	subscription.AutoRenewStatus = false
	return database.UpdateSubscription(subscription)
}

// handleDidRefund handles refund
func handleDidRefund(transactionInfo *models.TransactionInfo, projectID string) error {
	logging.Infof("Handling DID_REFUND - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	subscription.Status = "refunded"
	subscription.AutoRenewStatus = false
	return database.UpdateSubscription(subscription)
}

// handleExpired handles expiration
func handleExpired(transactionInfo *models.TransactionInfo, projectID string) error {
	logging.Infof("Handling EXPIRED - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	subscription.Status = "expired"
	subscription.AutoRenewStatus = false
	return database.UpdateSubscription(subscription)
}

// getPlanFromProductID determines plan from product ID
func getPlanFromProductID(productID string) string {
	switch productID {
	case "com.dailyzen.monthly", "monthly":
		return "monthly"
	case "com.dailyzen.yearly", "yearly":
		return "yearly"
	default:
		return "basic"
	}
}

