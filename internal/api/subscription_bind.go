package api

import (
	"net/http"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/pkg/logging"

	"github.com/gin-gonic/gin"
)

// BindAccountRequest represents bind account request
type BindAccountRequest struct {
	UserID string `json:"user_id" binding:"required"` // User ID to bind

	// iOS specific
	OriginalTransactionID string `json:"original_transaction_id,omitempty"` // iOS original transaction ID

	// Android specific
	PurchaseToken string `json:"purchase_token,omitempty"` // Android purchase token
}

// BindAccountResponse represents bind account response
type BindAccountResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// BindAccount binds user_id to a subscription
// POST /api/subscription/bind_account
// Used to bind user_id when webhook arrives before user verification
func BindAccount(c *gin.Context) {
	var req BindAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BindAccountResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate that at least one identifier is provided
	if req.OriginalTransactionID == "" && req.PurchaseToken == "" {
		c.JSON(http.StatusBadRequest, BindAccountResponse{
			Success: false,
			Message: "Either original_transaction_id (iOS) or purchase_token (Android) is required",
		})
		return
	}

	var subscription *models.Subscription
	var err error

	// Find subscription by identifier
	if req.OriginalTransactionID != "" {
		// iOS: Find by original_transaction_id
		// Note: We need project_id, but for binding we can search across all projects
		// or require project_id in request. For now, we'll search all projects.
		subscription, err = database.FindSubscriptionByOriginalTransactionID(req.OriginalTransactionID)
	} else {
		// Android: Find by purchase_token
		subscription, err = database.FindSubscriptionByPurchaseToken(req.PurchaseToken)
	}

	if err != nil {
		logging.Errorf("Failed to find subscription: %v", err)
		c.JSON(http.StatusNotFound, BindAccountResponse{
			Success: false,
			Message: "Subscription not found",
		})
		return
	}

	// Update user_id
	subscription.UserID = req.UserID
	if err := database.UpdateSubscription(subscription); err != nil {
		logging.Errorf("Failed to bind user_id: %v", err)
		c.JSON(http.StatusInternalServerError, BindAccountResponse{
			Success: false,
			Message: "Failed to bind account",
		})
		return
	}

	c.JSON(http.StatusOK, BindAccountResponse{
		Success: true,
		Message: "Account bound successfully",
	})
}

