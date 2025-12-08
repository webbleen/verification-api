package api

import (
	"encoding/json"
	"net/http"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/internal/services"
	"verification-api/pkg/logging"

	"github.com/gin-gonic/gin"
)

// GooglePlayNotification represents Google Play Real-Time Developer Notification
type GooglePlayNotification struct {
	Message struct {
		Data string `json:"data"` // Base64 encoded protobuf message
	} `json:"message"`
	SubscriptionNotification struct {
		NotificationType int    `json:"notificationType"` // 1=SUBSCRIPTION_RECOVERED, 2=SUBSCRIPTION_RENEWED, etc.
		PurchaseToken    string `json:"purchaseToken"`
		SubscriptionID   string `json:"subscriptionId"`
	} `json:"subscriptionNotification"`
}

// GooglePlayWebhookHandler handles Google Play Real-Time Developer Notifications
// POST /webhook/google
func GooglePlayWebhookHandler(c *gin.Context) {
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
	var notification GooglePlayNotification
	if err := json.Unmarshal(body, &notification); err != nil {
		logging.Errorf("Failed to parse Google Play notification: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid notification format",
		})
		return
	}

	// Extract purchase token and subscription ID
	purchaseToken := notification.SubscriptionNotification.PurchaseToken
	subscriptionID := notification.SubscriptionNotification.SubscriptionID
	notificationType := notification.SubscriptionNotification.NotificationType

	if purchaseToken == "" || subscriptionID == "" {
		logging.Errorf("Missing purchase_token or subscription_id in notification")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Missing required fields: purchase_token or subscription_id",
		})
		return
	}

	// TODO: Get project by package_name (need to extract from purchase token or use default)
	// For now, we'll need to query all projects or use a default
	projectService := services.NewProjectService()
	projects, err := projectService.GetAllProjects()
	if err != nil {
		logging.Errorf("Failed to get projects: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get projects",
		})
		return
	}

	// Find project by matching package_name (would need to query Google API to get package name)
	// For now, use first active project as fallback
	var project *models.Project
	if len(projects) > 0 {
		project = projects[0]
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "No active projects found",
		})
		return
	}

	// Query Google Play API to get latest subscription status
	verificationService := services.NewSubscriptionVerificationService()
	subscription, err := verificationService.VerifyGooglePlayPurchase(
		project.ProjectID,
		purchaseToken,
		subscriptionID,
		"", // user_id will be set when user verifies
	)

	if err != nil {
		logging.Errorf("Failed to verify Google Play purchase: %v", err)
		// Still return success to Google (we'll retry later)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Notification received, verification will be retried",
		})
		return
	}

	// Update subscription based on notification type
	switch notificationType {
	case 1: // SUBSCRIPTION_RECOVERED
		subscription.Status = "active"
	case 2: // SUBSCRIPTION_RENEWED
		subscription.Status = "active"
	case 3: // SUBSCRIPTION_CANCELED
		subscription.Status = "cancelled"
	case 4: // SUBSCRIPTION_PURCHASED
		subscription.Status = "active"
	case 5: // SUBSCRIPTION_ON_HOLD
		subscription.Status = "on_hold"
	case 6: // SUBSCRIPTION_IN_GRACE_PERIOD
		subscription.Status = "grace_period"
	case 7: // SUBSCRIPTION_RESTARTED
		subscription.Status = "active"
	case 8: // SUBSCRIPTION_PRICE_CHANGE_CONFIRMED
		// Status remains the same
	case 9: // SUBSCRIPTION_DEFERRED
		subscription.Status = "deferred"
	case 10: // SUBSCRIPTION_PAUSED
		subscription.Status = "paused"
	case 11: // SUBSCRIPTION_PAUSE_SCHEDULE_CHANGED
		// Status remains the same
	case 12: // SUBSCRIPTION_REVOKED
		subscription.Status = "revoked"
	case 13: // SUBSCRIPTION_EXPIRED
		subscription.Status = "expired"
	}

	// Update subscription in database
	if err := database.UpdateSubscription(subscription); err != nil {
		logging.Errorf("Failed to update subscription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update subscription",
		})
		return
	}

	// Notify App Backend via webhook if configured
	if project.WebhookCallbackURL != "" {
		go func() {
			webhookNotifier := services.NewWebhookNotifier()
			webhookNotifier.NotifyAppBackend(project.WebhookCallbackURL, project.WebhookSecret, subscription)
		}()
	}

	processingTime := time.Since(startTime)
	logging.Infof("Google Play notification processed - type: %d, subscription: %s, time: %v",
		notificationType, subscriptionID, processingTime)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification processed successfully",
	})
}
