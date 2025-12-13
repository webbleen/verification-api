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

var (
	// Global signature verifier instance
	signatureVerifier = services.NewSignatureVerifier()
	// Global replay protection instance
	replayProtection = services.NewReplayProtection()
)

// processAppStoreNotification processes App Store notification
// If body is nil, it will be read from the context
func processAppStoreNotification(environment string, c *gin.Context, body []byte, signatureHeader string) {
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

	// Verify signature if present
	if signatureHeader != "" {
		if err := signatureVerifier.VerifyNotification(body, signatureHeader); err != nil {
			logging.Errorf("Signature verification failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Signature verification failed",
			})
			return
		}
		logging.Infof("Signature verification passed")
	} else {
		logging.Infof("No signature header present, skipping verification")
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
	logging.Infof("Parsed notification - type: %s, bundle_id: %s, environment: %s, data_version: %s, uuid: %s",
		notification.NotificationType, notification.Data.BundleID, notification.Data.Environment, notification.DataVersion, notification.NotificationUUID)

	// Handle heartbeat
	if notification.NotificationType == "" {
		logging.Infof("AppStore heartbeat - environment: %s", environment)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"status":  "heartbeat_ok",
		})
		return
	}

	// Check for replay attacks
	if replayProtection.IsReplay(notification.NotificationUUID, notification.SignedDate) {
		logging.Errorf("Replay attack detected - notification_uuid: %s, signed_date: %d", notification.NotificationUUID, notification.SignedDate)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Duplicate notification detected",
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

	logging.Infof("Parsed transaction info - transaction_id: %s, original_transaction_id: %s, product_id: %s, app_account_token: %s",
		transactionInfo.TransactionID, transactionInfo.OriginalTransactionID, transactionInfo.ProductID, transactionInfo.AppAccountToken)

	// Note: appAccountToken is a UUID set by the client during purchase (applicationUserName parameter)
	// We need to query App Backend to get the actual device_id (user_id) from appAccountToken
	// If appAccountToken is empty, we cannot determine user_id (should not happen in normal flow)

	// Query device_id from App Backend using appAccountToken
	if transactionInfo.AppAccountToken != "" && project.WebhookCallbackURL != "" {
		// Extract base URL from webhook callback URL (e.g., https://api.example.com/webhooks/unionhub -> https://api.example.com)
		baseURL := extractBaseURL(project.WebhookCallbackURL)
		if baseURL != "" {
			deviceID, err := queryDeviceIDFromAppBackend(baseURL, transactionInfo.AppAccountToken)
			if err != nil {
				logging.Infof("Failed to query device_id from App Backend: %v, will use appAccountToken (UUID) as user_id", err)
				// Fallback: use appAccountToken as user_id (UUID format)
				// This is acceptable as appAccountToken is already a UUID
			} else if deviceID != "" {
				logging.Infof("Resolved device_id from appAccountToken - AppAccountToken: %s, DeviceID: %s", transactionInfo.AppAccountToken, deviceID)
				// Replace appAccountToken with actual device_id
				transactionInfo.AppAccountToken = deviceID
			}
		}
	}

	// Handle notification by type
	subscription, err := handleNotificationByType(notification.NotificationType, transactionInfo, project.ProjectID, notification.Data.Environment)
	if err != nil {
		logging.Errorf("Failed to handle notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to process notification",
		})
		return
	}

	// Notify App Backend via webhook if configured
	if subscription != nil && project.WebhookCallbackURL != "" {
		go func() {
			webhookNotifier := services.NewWebhookNotifier()
			webhookNotifier.NotifyAppBackend(project.WebhookCallbackURL, project.WebhookSecret, subscription)
		}()
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
	// Get signature header
	signature := c.GetHeader("X-Apple-Notification-Signature")
	if signature != "" {
		logging.Infof("Received Apple production webhook with signature: %s...", signature[:min(len(signature), 20)])
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
	processAppStoreNotification("production", c, body, signature)
}

// AppStoreSandboxWebhookHandler handles sandbox environment webhook
// POST /webhook/apple/sandbox
func AppStoreSandboxWebhookHandler(c *gin.Context) {
	// Get signature header
	signature := c.GetHeader("X-Apple-Notification-Signature")
	if signature != "" {
		logging.Infof("Received Apple sandbox webhook with signature: %s...", signature[:min(len(signature), 20)])
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
	processAppStoreNotification("sandbox", c, body, signature)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	// Debug: Log all claim keys to help identify appAccountToken field name
	claimKeys := make([]string, 0, len(claims))
	for k := range claims {
		claimKeys = append(claimKeys, k)
	}
	logging.Infof("JWT claims keys: %v", claimKeys)

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

	// Extract appAccountToken (user_id passed from client during purchase)
	// Apple stores this as a UUID string in the JWT claims
	// Try different possible field names
	var appAccountToken string
	if aat, ok := claims["appAccountToken"].(string); ok {
		appAccountToken = aat
	} else if aat, ok := claims["app_account_token"].(string); ok {
		appAccountToken = aat
	} else if aat, ok := claims["applicationUsername"].(string); ok {
		appAccountToken = aat
	} else if aat, ok := claims["application_username"].(string); ok {
		appAccountToken = aat
	}

	if appAccountToken != "" {
		transactionInfo.AppAccountToken = appAccountToken
		logging.Infof("Extracted appAccountToken from transaction: %s", transactionInfo.AppAccountToken)
	} else {
		logging.Infof("No appAccountToken found in JWT claims. Available keys: %v", claimKeys)
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
// Returns the updated subscription and error
func handleNotificationByType(notificationType string, transactionInfo *models.TransactionInfo, projectID, environment string) (*models.Subscription, error) {
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
		return nil, nil
	}
}

// handleInitialBuy handles initial purchase
func handleInitialBuy(transactionInfo *models.TransactionInfo, projectID, environment string) (*models.Subscription, error) {
	logging.Infof("Handling INITIAL_BUY - transaction: %s, original_transaction: %s, product: %s, app_account_token: %s",
		transactionInfo.TransactionID, transactionInfo.OriginalTransactionID, transactionInfo.ProductID, transactionInfo.AppAccountToken)

	// Use appAccountToken as user_id (set by client during purchase)
	userID := transactionInfo.AppAccountToken
	if userID == "" {
		logging.Infof("No appAccountToken in transaction - user_id will be empty. This should not happen if client sets applicationUserName during purchase.")
	}

	// Find existing subscription by original transaction ID
	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		// Create new subscription
		subscription = &models.Subscription{
			ProjectID:             projectID,
			AppAccountToken:       userID, // Use appAccountToken if available
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
			return nil, fmt.Errorf("failed to create subscription: %w", err)
		}

		if userID != "" {
			logging.Infof("Created new subscription with uuid from appAccountToken - transaction: %s, original_transaction: %s, uuid: %s",
				transactionInfo.TransactionID, transactionInfo.OriginalTransactionID, userID)
		} else {
			logging.Infof("Created new subscription - transaction: %s, original_transaction: %s",
				transactionInfo.TransactionID, transactionInfo.OriginalTransactionID)
		}
		return subscription, nil
	}

	// Update existing subscription
	// If subscription has no appAccountToken but we have one, bind it
	if subscription.AppAccountToken == "" && userID != "" {
		subscription.AppAccountToken = userID
		logging.Infof("Binding appAccountToken to existing subscription - original_transaction: %s, app_account_token: %s",
			transactionInfo.OriginalTransactionID, userID)
	}

	subscription.Status = "active"
	subscription.ExpiresDate = time.Unix(transactionInfo.ExpiresDateMS/1000, 0)
	subscription.AutoRenewStatus = transactionInfo.AutoRenewStatus == 1

	if err := database.UpdateSubscription(subscription); err != nil {
		logging.Errorf("Failed to update subscription: %v", err)
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	logging.Infof("Updated existing subscription - transaction: %s, original_transaction: %s",
		transactionInfo.TransactionID, transactionInfo.OriginalTransactionID)
	return subscription, nil
}

// handleDidRenew handles renewal
func handleDidRenew(transactionInfo *models.TransactionInfo, projectID string) (*models.Subscription, error) {
	logging.Infof("Handling DID_RENEW - transaction: %s, app_account_token: %s", transactionInfo.TransactionID, transactionInfo.AppAccountToken)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// If subscription has no appAccountToken but we have one, bind it
	if subscription.AppAccountToken == "" && transactionInfo.AppAccountToken != "" {
		subscription.AppAccountToken = transactionInfo.AppAccountToken
		logging.Infof("Binding appAccountToken during renewal - original_transaction: %s, app_account_token: %s",
			transactionInfo.OriginalTransactionID, transactionInfo.AppAccountToken)
	}

	subscription.Status = "active"
	subscription.ExpiresDate = time.Unix(transactionInfo.ExpiresDateMS/1000, 0)
	subscription.AutoRenewStatus = transactionInfo.AutoRenewStatus == 1
	if err := database.UpdateSubscription(subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

// handleDidFailToRenew handles failed renewal
func handleDidFailToRenew(transactionInfo *models.TransactionInfo, projectID string) (*models.Subscription, error) {
	logging.Infof("Handling DID_FAIL_TO_RENEW - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// If subscription has no appAccountToken but we have one, bind it
	if subscription.AppAccountToken == "" && transactionInfo.AppAccountToken != "" {
		subscription.AppAccountToken = transactionInfo.AppAccountToken
		logging.Infof("Binding appAccountToken - original_transaction: %s, app_account_token: %s",
			transactionInfo.OriginalTransactionID, transactionInfo.AppAccountToken)
	}

	subscription.Status = "failed"
	subscription.AutoRenewStatus = false
	if err := database.UpdateSubscription(subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

// handleDidCancel handles cancellation
func handleDidCancel(transactionInfo *models.TransactionInfo, projectID string) (*models.Subscription, error) {
	logging.Infof("Handling DID_CANCEL - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// If subscription has no appAccountToken but we have one, bind it
	if subscription.AppAccountToken == "" && transactionInfo.AppAccountToken != "" {
		subscription.AppAccountToken = transactionInfo.AppAccountToken
		logging.Infof("Binding appAccountToken - original_transaction: %s, app_account_token: %s",
			transactionInfo.OriginalTransactionID, transactionInfo.AppAccountToken)
	}

	subscription.Status = "cancelled"
	subscription.AutoRenewStatus = false
	if err := database.UpdateSubscription(subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

// handleDidRefund handles refund
func handleDidRefund(transactionInfo *models.TransactionInfo, projectID string) (*models.Subscription, error) {
	logging.Infof("Handling DID_REFUND - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// If subscription has no appAccountToken but we have one, bind it
	if subscription.AppAccountToken == "" && transactionInfo.AppAccountToken != "" {
		subscription.AppAccountToken = transactionInfo.AppAccountToken
		logging.Infof("Binding appAccountToken - original_transaction: %s, app_account_token: %s",
			transactionInfo.OriginalTransactionID, transactionInfo.AppAccountToken)
	}

	subscription.Status = "refunded"
	subscription.AutoRenewStatus = false
	if err := database.UpdateSubscription(subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

// handleExpired handles expiration
func handleExpired(transactionInfo *models.TransactionInfo, projectID string) (*models.Subscription, error) {
	logging.Infof("Handling EXPIRED - transaction: %s", transactionInfo.TransactionID)

	subscription, err := database.GetSubscriptionByOriginalTransactionID(projectID, transactionInfo.OriginalTransactionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// If subscription has no appAccountToken but we have one, bind it
	if subscription.AppAccountToken == "" && transactionInfo.AppAccountToken != "" {
		subscription.AppAccountToken = transactionInfo.AppAccountToken
		logging.Infof("Binding appAccountToken - original_transaction: %s, app_account_token: %s",
			transactionInfo.OriginalTransactionID, transactionInfo.AppAccountToken)
	}

	subscription.Status = "expired"
	subscription.AutoRenewStatus = false
	if err := database.UpdateSubscription(subscription); err != nil {
		return nil, err
	}
	return subscription, nil
}

// extractBaseURL extracts base URL from webhook callback URL
// e.g., https://api.example.com/webhooks/unionhub -> https://api.example.com
func extractBaseURL(webhookURL string) string {
	// Simple extraction: remove /webhooks/unionhub or similar paths
	if strings.Contains(webhookURL, "/webhooks/") {
		parts := strings.Split(webhookURL, "/webhooks/")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	// If no /webhooks/ found, try to extract base URL by removing last path segment
	lastSlash := strings.LastIndex(webhookURL, "/")
	if lastSlash > 0 {
		// Find the protocol part (http:// or https://)
		protocolEnd := strings.Index(webhookURL, "://")
		if protocolEnd > 0 {
			// Find the next slash after protocol
			pathStart := strings.Index(webhookURL[protocolEnd+3:], "/")
			if pathStart > 0 {
				return webhookURL[:protocolEnd+3+pathStart]
			}
		}
		return webhookURL[:lastSlash]
	}
	return webhookURL
}

// queryDeviceIDFromAppBackend queries App Backend to get device_id from app_account_token
func queryDeviceIDFromAppBackend(baseURL, appAccountToken string) (string, error) {
	url := fmt.Sprintf("%s/api/app-account-token/device-id?app_account_token=%s", baseURL, appAccountToken)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to query App Backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("App Backend returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		return "", fmt.Errorf("App Backend query failed")
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		if deviceID, ok := data["device_id"].(string); ok {
			return deviceID, nil
		}
	}

	return "", fmt.Errorf("device_id not found in response")
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
