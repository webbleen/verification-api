package api

import (
	"net/http"
	"time"
	"verification-api/internal/models"
	"verification-api/internal/services"

	"github.com/gin-gonic/gin"
)

// VerifySubscriptionRequest represents verify subscription request
type VerifySubscriptionRequest struct {
	Platform    string `json:"platform" binding:"required,oneof=ios android"` // ios or android
	ReceiptData string `json:"receipt_data" binding:"required"`                // Base64 receipt (iOS) or purchase token (Android)
	UserID      string `json:"user_id" binding:"required"`                     // User ID from the app
	AppID       string `json:"app_id" binding:"required"`                       // Bundle ID (iOS) or Package Name (Android)
}

// VerifySubscriptionResponse represents verify subscription response
type VerifySubscriptionResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	IsActive   bool   `json:"is_active"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	Plan       string `json:"plan,omitempty"`
	ProductID  string `json:"product_id,omitempty"`
}

// VerifySubscription verifies subscription receipt/token
// POST /api/subscription/verify
func VerifySubscription(c *gin.Context) {
	var req VerifySubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, VerifySubscriptionResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Get project by app_id (bundle_id or package_name)
	projectService := services.NewProjectService()
	var project *models.Project
	var err error

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

	// Verify receipt/token
	verificationService := services.NewSubscriptionVerificationService()
	var subscription *models.Subscription

	if req.Platform == "ios" {
		subscription, err = verificationService.VerifyAppleReceipt(project.ProjectID, req.ReceiptData, req.UserID)
	} else {
		// For Android, extract product_id from request if needed
		// For now, we'll use a placeholder
		subscription, err = verificationService.VerifyGooglePlayPurchase(project.ProjectID, req.ReceiptData, "", req.UserID)
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
		Success:   true,
		Message:   "Subscription verified successfully",
		IsActive:  isActive,
		ExpiresAt: subscription.ExpiresDate.Format(time.RFC3339),
		Plan:      subscription.Plan,
		ProductID: subscription.ProductID,
	})
}

