package api

import (
	"net/http"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/internal/services"
	"verification-api/pkg/logging"

	"github.com/gin-gonic/gin"
)

// TransactionInfo represents a transaction to restore
type TransactionInfo struct {
	SignedTransaction string `json:"signed_transaction,omitempty"` // JWT signed transaction (iOS)
	TransactionID     string `json:"transaction_id,omitempty"`     // Transaction ID (iOS)
	ProductID         string `json:"product_id,omitempty"`          // Product ID
}

// RestoreSubscriptionRequest represents restore subscription request
// Supports two modes:
// 1. Active restore: Client provides transaction list, UnionHub verifies each one
// 2. Passive restore: Client only provides user_id, UnionHub looks up from database
type RestoreSubscriptionRequest struct {
	UserID      string           `json:"user_id" binding:"required"`      // User ID from the app
	AppID       string           `json:"app_id,omitempty"`                // Bundle ID (iOS) or Package Name (Android) - optional if transactions provided
	Platform    string           `json:"platform" binding:"required,oneof=ios android"` // ios or android
	Transactions []TransactionInfo `json:"transactions,omitempty"`        // List of transactions to verify (for active restore)
}

// SubscriptionInfo represents a subscription in restore response
type SubscriptionInfo struct {
	IsActive    bool   `json:"is_active"`
	Status      string `json:"status"`
	ExpiresDate string `json:"expires_date,omitempty"`
	ProductID   string `json:"product_id,omitempty"`
	AutoRenew   bool   `json:"auto_renew,omitempty"`
}

// RestoreSubscriptionResponse represents restore subscription response
type RestoreSubscriptionResponse struct {
	Success      bool              `json:"success"`
	Message      string            `json:"message"`
	Subscriptions []SubscriptionInfo `json:"subscriptions,omitempty"` // List of all active subscriptions
	// Legacy fields (for backward compatibility)
	IsActive  bool   `json:"is_active,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	ProductID string `json:"product_id,omitempty"`
}

// RestoreSubscription restores subscription by verifying transactions
// POST /api/subscription/restore
// Supports two modes:
// 1. Active restore: Client provides transaction list, UnionHub actively verifies each transaction
// 2. Passive restore: Client only provides user_id, UnionHub looks up from database
func RestoreSubscription(c *gin.Context) {
	var req RestoreSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RestoreSubscriptionResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	projectService := services.NewProjectService()
	var project *models.Project
	var err error

	// Get project - try from app_id first, or extract from first transaction
	if req.AppID != "" {
		if req.Platform == "ios" {
			project, err = projectService.GetProjectByBundleID(req.AppID)
		} else {
			project, err = projectService.GetProjectByPackageName(req.AppID)
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, RestoreSubscriptionResponse{
				Success: false,
				Message: "App not found: " + err.Error(),
			})
			return
		}
	} else if len(req.Transactions) > 0 && req.Transactions[0].SignedTransaction != "" {
		// Try to extract bundle_id from first transaction
		// TODO: Extract bundle_id from signed_transaction JWT
		c.JSON(http.StatusBadRequest, RestoreSubscriptionResponse{
			Success: false,
			Message: "app_id is required when transactions are not provided or bundle_id cannot be extracted",
		})
		return
	} else {
		c.JSON(http.StatusBadRequest, RestoreSubscriptionResponse{
			Success: false,
			Message: "app_id is required",
		})
		return
	}

	verificationService := services.NewSubscriptionVerificationService()
	var activeSubscriptions []SubscriptionInfo

	// Mode 1: Active restore - verify each transaction provided by client
	if len(req.Transactions) > 0 {
		logging.Infof("Active restore: verifying %d transactions for user %s", len(req.Transactions), req.UserID)
		
		for _, tx := range req.Transactions {
			if req.Platform == "ios" {
				// Verify iOS transaction
				subscription, err := verificationService.VerifyAppleTransaction(
					project.ProjectID,
					tx.SignedTransaction,
					tx.TransactionID,
					tx.ProductID,
					req.UserID,
				)
				
				if err != nil {
					logging.Errorf("Failed to verify transaction %s: %v", tx.TransactionID, err)
					continue
				}
				
				// Check if subscription is active
				isActive := subscription.Status == "active" && subscription.ExpiresDate.After(time.Now())
				
				activeSubscriptions = append(activeSubscriptions, SubscriptionInfo{
					IsActive:    isActive,
					Status:      subscription.Status,
					ExpiresDate: subscription.ExpiresDate.Format(time.RFC3339),
					ProductID:   subscription.ProductID,
					AutoRenew:   subscription.AutoRenewStatus,
				})
			} else {
				// Android restore - TODO: implement when Google Play restore is needed
				c.JSON(http.StatusBadRequest, RestoreSubscriptionResponse{
					Success: false,
					Message: "Android restore with transaction list not yet implemented",
				})
				return
			}
		}
	} else {
		// Mode 2: Passive restore - look up from database
		logging.Infof("Passive restore: looking up subscriptions for user %s", req.UserID)
		
		subscriptions, err := database.GetUserSubscriptions(project.ProjectID, req.UserID)
		if err != nil {
			c.JSON(http.StatusNotFound, RestoreSubscriptionResponse{
				Success: false,
				Message: "No subscription found for this user",
			})
			return
		}
		
		// Filter active subscriptions and convert to response format
		for _, sub := range subscriptions {
			isActive := sub.Status == "active" && sub.ExpiresDate.After(time.Now())
			
			activeSubscriptions = append(activeSubscriptions, SubscriptionInfo{
				IsActive:    isActive,
				Status:      sub.Status,
				ExpiresDate: sub.ExpiresDate.Format(time.RFC3339),
				ProductID:   sub.ProductID,
				AutoRenew:   sub.AutoRenewStatus,
			})
		}
	}

	// If no active subscriptions found
	if len(activeSubscriptions) == 0 {
		c.JSON(http.StatusOK, RestoreSubscriptionResponse{
			Success:      true,
			Message:      "No active subscriptions found",
			Subscriptions: []SubscriptionInfo{},
		})
		return
	}

	// Find the most recent active subscription (for backward compatibility)
	var latestActive *SubscriptionInfo
	for i := range activeSubscriptions {
		if activeSubscriptions[i].IsActive {
			if latestActive == nil {
				latestActive = &activeSubscriptions[i]
			}
		}
	}

	response := RestoreSubscriptionResponse{
		Success:       true,
		Message:       "Subscription restored successfully",
		Subscriptions: activeSubscriptions,
	}

	// Legacy fields for backward compatibility
	if latestActive != nil {
		response.IsActive = latestActive.IsActive
		response.ExpiresAt = latestActive.ExpiresDate
		response.ProductID = latestActive.ProductID
	}

	c.JSON(http.StatusOK, response)
}

