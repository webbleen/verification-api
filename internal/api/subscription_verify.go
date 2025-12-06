package api

import (
	"net/http"
	"time"
	"verification-api/internal/models"
	"verification-api/internal/services"

	"github.com/gin-gonic/gin"
)

// VerifySubscriptionRequest represents verify subscription request
// Supports platform-specific fields as per industry standards
type VerifySubscriptionRequest struct {
	Platform string `json:"platform" binding:"required,oneof=ios android"` // ios or android
	UserID   string `json:"user_id" binding:"required"`                     // User ID from the app
	ProductID string `json:"product_id" binding:"required"`                 // Product ID (required for both platforms)

	// iOS specific fields
	SignedTransaction string `json:"signed_transaction,omitempty"` // JWT signed transaction (iOS)
	TransactionID     string `json:"transaction_id,omitempty"`     // Transaction ID (iOS)

	// Android specific fields
	PurchaseToken string `json:"purchase_token,omitempty"` // Purchase token (Android)

	// Legacy support (deprecated, use platform-specific fields)
	ReceiptData string `json:"receipt_data,omitempty"` // Legacy: Base64 receipt (iOS) or purchase token (Android)
	AppID       string `json:"app_id,omitempty"`        // Legacy: Bundle ID (iOS) or Package Name (Android)
}

// VerifySubscriptionResponse represents verify subscription response
type VerifySubscriptionResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	IsActive    bool   `json:"is_active"`
	Platform    string `json:"platform,omitempty"`     // Platform: ios or android
	ExpiresDate string `json:"expires_date,omitempty"` // ISO 8601 format
	Plan        string `json:"plan,omitempty"`
	ProductID   string `json:"product_id,omitempty"`
	AutoRenew   bool   `json:"auto_renew,omitempty"`

	// Legacy support (deprecated)
	ExpiresAt string `json:"expires_at,omitempty"` // Deprecated: use expires_date
}

// VerifySubscription verifies subscription receipt/token
// POST /api/subscription/verify
// Supports both new platform-specific format and legacy format
func VerifySubscription(c *gin.Context) {
	var req VerifySubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate platform-specific fields
	if req.Platform == "ios" {
		// iOS requires signed_transaction or transaction_id (or legacy receipt_data)
		if req.SignedTransaction == "" && req.TransactionID == "" && req.ReceiptData == "" {
			c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
				Success: false,
				Message: "iOS requires signed_transaction or transaction_id",
			})
			return
		}
	} else if req.Platform == "android" {
		// Android requires purchase_token (or legacy receipt_data)
		if req.PurchaseToken == "" && req.ReceiptData == "" {
			c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
				Success: false,
				Message: "Android requires purchase_token",
			})
			return
		}
	}

	// Get project - try to get from bundle_id/package_name if available
	// Otherwise, we'll need to extract from signed_transaction or use default
	projectService := services.NewProjectService()
	var project *models.Project
	var err error

	// Try to get project from app_id (legacy) or extract from transaction
	if req.AppID != "" {
		if req.Platform == "ios" {
			project, err = projectService.GetProjectByBundleID(req.AppID)
		} else {
			project, err = projectService.GetProjectByPackageName(req.AppID)
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
				Success: false,
				Message: "App not found: " + err.Error(),
			})
			return
		}
	} else {
		// TODO: Extract bundle_id/package_name from signed_transaction or use default project
		// For now, return error if app_id is not provided
		c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
			Success: false,
			Message: "app_id is required (or extract from signed_transaction)",
		})
		return
	}

	// Verify receipt/token
	verificationService := services.NewSubscriptionVerificationService()
	var subscription *models.Subscription

	if req.Platform == "ios" {
		// Use new format if available, fallback to legacy
		if req.SignedTransaction != "" || req.TransactionID != "" {
			subscription, err = verificationService.VerifyAppleTransaction(
				project.ProjectID,
				req.SignedTransaction,
				req.TransactionID,
				req.ProductID,
				req.UserID,
			)
		} else {
			// Legacy format
			subscription, err = verificationService.VerifyAppleReceipt(project.ProjectID, req.ReceiptData, req.UserID)
		}
	} else {
		// Android
		purchaseToken := req.PurchaseToken
		if purchaseToken == "" {
			purchaseToken = req.ReceiptData // Legacy support
		}
		subscription, err = verificationService.VerifyGooglePlayPurchase(
			project.ProjectID,
			purchaseToken,
			req.ProductID,
			req.UserID,
		)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
			Success: false,
			Message: "Verification failed: " + err.Error(),
		})
		return
	}

	// Check if subscription is active
	isActive := subscription.Status == "active" && subscription.ExpiresDate.After(time.Now())

	c.JSON(http.StatusOK, VerifySubscriptionResponse{
		Success:     true,
		Message:     "Subscription verified successfully",
		IsActive:    isActive,
		Platform:    subscription.Platform,
		ExpiresDate: subscription.ExpiresDate.Format(time.RFC3339),
		ExpiresAt:   subscription.ExpiresDate.Format(time.RFC3339), // Legacy support
		Plan:        subscription.Plan,
		ProductID:   subscription.ProductID,
		AutoRenew:   subscription.AutoRenewStatus,
	})
}

