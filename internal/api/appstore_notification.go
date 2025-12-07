package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/internal/services"
	"verification-api/pkg/logging"

	"github.com/gin-gonic/gin"
)

// processAppStoreNotification processes App Store notification
// If body is nil, it will be read from the context
func processAppStoreNotification(environment string, c *gin.Context, body []byte) {
	startTime := time.Now()

	// Read raw body if not provided
	var err error
	if body == nil {
		body, err = c.GetRawData()
		if err != nil {
			logging.Errorf("Failed to read request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Failed to read request body",
			})
			return
		}
	}

	// Check if body is empty
	if len(body) == 0 {
		logging.Errorf("Empty request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Empty request body",
		})
		return
	}

	// Parse the wrapper to get signedPayload
	var wrapper models.AppStoreNotificationWrapper
	if err := json.Unmarshal(body, &wrapper); err != nil {
		logging.Errorf("Failed to parse notification wrapper: %v, body length: %d", err, len(body))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid notification format",
		})
		return
	}

	if wrapper.SignedPayload == "" {
		logging.Errorf("signedPayload is empty in notification")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "signedPayload is missing",
		})
		return
	}

	// Parse JWT manually to skip signature verification
	// JWT format: header.payload.signature
	parts := strings.Split(wrapper.SignedPayload, ".")
	if len(parts) != 3 {
		logging.Errorf("Invalid JWT format: expected 3 parts, got %d", len(parts))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid JWT format",
		})
		return
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		logging.Errorf("Failed to decode JWT payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to decode JWT payload",
		})
		return
	}

	// Parse notification from payload
	var notification models.AppStoreNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		previewLen := 500
		if len(payload) < previewLen {
			previewLen = len(payload)
		}
		logging.Errorf("Failed to unmarshal notification from JWT payload: %v, payload preview: %s", err, string(payload[:previewLen]))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to parse notification from JWT",
		})
		return
	}

	// Log parsed notification details
	logging.Infof("Parsed notification - type: %s, bundle_id: %s, environment: %s, data_version: %s", 
		notification.NotificationType, notification.Data.BundleID, notification.Data.Environment, notification.DataVersion)

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

	logging.Infof("Found project: %s (project_id: %s)", project.ProjectName, project.ProjectID)

	// Parse transaction info from JWT
	transactionInfo, err := parseTransactionInfo(notification.Data.SignedTransactionInfo)
	if err != nil {
		logging.Errorf("Failed to parse transaction info: %v, signed_transaction_info length: %d", err, len(notification.Data.SignedTransactionInfo))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to parse transaction info",
		})
		return
	}

	logging.Infof("Parsed transaction info - transaction_id: %s, original_transaction_id: %s, product_id: %s",
		transactionInfo.TransactionID, transactionInfo.OriginalTransactionID, transactionInfo.ProductID)

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

// AppStoreProductionWebhookHandler handles production environment webhook
// POST /webhook/apple/production
func AppStoreProductionWebhookHandler(c *gin.Context) {
	// Verify JWT signature if present
	signature := c.GetHeader("X-Apple-Notification-Signature")
	if signature != "" {
		// TODO: Verify JWT signature using Apple's public keys
		// For now, we'll process the notification
		logging.Infof("Received Apple production webhook with signature: %s", signature[:20]+"...")
	}

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

	// Process notification with production environment
	processAppStoreNotification("production", c, body)
}

// AppStoreSandboxWebhookHandler handles sandbox environment webhook
// POST /webhook/apple/sandbox
func AppStoreSandboxWebhookHandler(c *gin.Context) {
	// Verify JWT signature if present
	signature := c.GetHeader("X-Apple-Notification-Signature")
	if signature != "" {
		// TODO: Verify JWT signature using Apple's public keys
		// For now, we'll process the notification
		logging.Infof("Received Apple sandbox webhook with signature: %s", signature[:20]+"...")
	}

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

	// Process notification with sandbox environment
	processAppStoreNotification("sandbox", c, body)
}

// parseTransactionInfo parses transaction info from JWT string
// signedTransactionInfo is a JWT token from Apple App Store Server Notifications V2
func parseTransactionInfo(signedTransactionInfo string) (*models.TransactionInfo, error) {
	if signedTransactionInfo == "" {
		return nil, fmt.Errorf("signed_transaction_info is empty")
	}

	// Parse JWT manually to skip signature verification
	// JWT format: header.payload.signature
	parts := strings.Split(signedTransactionInfo, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse transaction info from payload
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT payload: %w", err)
	}

	// Extract transaction information from claims
	transactionInfo := &models.TransactionInfo{}

	if tid, ok := claims["transactionId"].(string); ok {
		transactionInfo.TransactionID = tid
	}

	if otid, ok := claims["originalTransactionId"].(string); ok {
		transactionInfo.OriginalTransactionID = otid
	}

	if pid, ok := claims["productId"].(string); ok {
		transactionInfo.ProductID = pid
	}

	// Handle purchaseDate (can be int64 or float64 in JSON)
	if pd, ok := claims["purchaseDate"]; ok {
		switch v := pd.(type) {
		case float64:
			transactionInfo.PurchaseDateMS = int64(v)
		case int64:
			transactionInfo.PurchaseDateMS = v
		case int:
			transactionInfo.PurchaseDateMS = int64(v)
		}
	}

	// Handle expiresDate (can be int64 or float64 in JSON)
	if ed, ok := claims["expiresDate"]; ok {
		switch v := ed.(type) {
		case float64:
			transactionInfo.ExpiresDateMS = int64(v)
		case int64:
			transactionInfo.ExpiresDateMS = v
		case int:
			transactionInfo.ExpiresDateMS = int64(v)
		}
	}

	// Handle autoRenewStatus (can be int or float64 in JSON)
	if ars, ok := claims["autoRenewStatus"]; ok {
		switch v := ars.(type) {
		case float64:
			transactionInfo.AutoRenewStatus = int(v)
		case int64:
			transactionInfo.AutoRenewStatus = int(v)
		case int:
			transactionInfo.AutoRenewStatus = v
		}
	}

	if env, ok := claims["environment"].(string); ok {
		transactionInfo.Environment = env
	}

	// Validate required fields
	if transactionInfo.TransactionID == "" {
		return nil, fmt.Errorf("transaction_id is missing in JWT claims")
	}

	if transactionInfo.OriginalTransactionID == "" {
		return nil, fmt.Errorf("original_transaction_id is missing in JWT claims")
	}

	return transactionInfo, nil
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
	logging.Infof("Handling INITIAL_BUY - transaction: %s, original_transaction: %s, product: %s", 
		transactionInfo.TransactionID, transactionInfo.OriginalTransactionID, transactionInfo.ProductID)

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
		
		if err := database.CreateSubscription(subscription); err != nil {
			logging.Errorf("Failed to create subscription: %v", err)
			return fmt.Errorf("failed to create subscription: %w", err)
		}
		
		logging.Infof("Created new subscription - transaction: %s, original_transaction: %s", 
			transactionInfo.TransactionID, transactionInfo.OriginalTransactionID)
		return nil
	}

	// Update existing subscription
	subscription.Status = "active"
	subscription.ExpiresDate = time.Unix(transactionInfo.ExpiresDateMS/1000, 0)
	subscription.AutoRenewStatus = transactionInfo.AutoRenewStatus == 1
	
	if err := database.UpdateSubscription(subscription); err != nil {
		logging.Errorf("Failed to update subscription: %v", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	
	logging.Infof("Updated existing subscription - transaction: %s, original_transaction: %s", 
		transactionInfo.TransactionID, transactionInfo.OriginalTransactionID)
	return nil
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

