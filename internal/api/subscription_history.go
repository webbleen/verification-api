package api

import (
	"net/http"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/internal/services"

	"github.com/gin-gonic/gin"
)

// SubscriptionHistoryItem represents a subscription history item
type SubscriptionHistoryItem struct {
	ID                  uint      `json:"id"`
	AppAccountToken     string    `json:"app_account_token"`
	Platform            string    `json:"platform"`
	Plan                string    `json:"plan"`
	Status              string    `json:"status"`
	ProductID           string    `json:"product_id"`
	TransactionID       string    `json:"transaction_id"`
	OriginalTransactionID string  `json:"original_transaction_id"`
	PurchaseDate        time.Time `json:"purchase_date"`
	ExpiresDate         time.Time `json:"expires_date"`
	AutoRenew           bool      `json:"auto_renew"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// SubscriptionHistoryResponse represents subscription history response
type SubscriptionHistoryResponse struct {
	Success      bool                     `json:"success"`
	Message      string                   `json:"message,omitempty"`
	Subscriptions []SubscriptionHistoryItem `json:"subscriptions,omitempty"`
}

// GetSubscriptionHistory gets subscription history for a user
// GET /api/subscription/history?user_id=xxx&app_id=yyy&platform=ios
func GetSubscriptionHistory(c *gin.Context) {
	userID := c.Query("user_id")
	appID := c.Query("app_id")
	platform := c.DefaultQuery("platform", "ios")

	if userID == "" {
		c.JSON(http.StatusBadRequest, SubscriptionHistoryResponse{
			Success: false,
			Message: "user_id is required",
		})
		return
	}

	// Get project by app_id
	var project *models.Project
	var err error

	if appID != "" {
		projectService := services.NewProjectService()
		if platform == "ios" {
			project, err = projectService.GetProjectByBundleID(appID)
		} else {
			project, err = projectService.GetProjectByPackageName(appID)
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, SubscriptionHistoryResponse{
				Success: false,
				Message: "App not found: " + err.Error(),
			})
			return
		}
	}

	// Get subscription history
	var subscriptions []models.Subscription
	if project != nil {
		subscriptions, err = database.GetUserSubscriptions(project.ProjectID, userID)
	} else {
		// If no app_id provided, get all subscriptions for user (across all projects)
		subscriptions, err = database.GetAllUserSubscriptions(userID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, SubscriptionHistoryResponse{
			Success: false,
			Message: "Failed to get subscription history: " + err.Error(),
		})
		return
	}

	// Convert to response format
	historyItems := make([]SubscriptionHistoryItem, len(subscriptions))
	for i, sub := range subscriptions {
		historyItems[i] = SubscriptionHistoryItem{
			ID:                  sub.ID,
			AppAccountToken:     sub.AppAccountToken,
			Platform:            sub.Platform,
			Plan:                sub.Plan,
			Status:              sub.Status,
			ProductID:           sub.ProductID,
			TransactionID:       sub.TransactionID,
			OriginalTransactionID: sub.OriginalTransactionID,
			PurchaseDate:        sub.PurchaseDate,
			ExpiresDate:         sub.ExpiresDate,
			AutoRenew:           sub.AutoRenewStatus,
			CreatedAt:           sub.CreatedAt,
			UpdatedAt:           sub.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, SubscriptionHistoryResponse{
		Success:      true,
		Subscriptions: historyItems,
	})
}

