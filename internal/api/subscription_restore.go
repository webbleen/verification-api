package api

import (
	"net/http"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/services"

	"github.com/gin-gonic/gin"
)

// RestoreSubscriptionRequest represents restore subscription request
type RestoreSubscriptionRequest struct {
	UserID string `json:"user_id" binding:"required"` // User ID from the app
	AppID  string `json:"app_id" binding:"required"`  // Bundle ID (iOS) or Package Name (Android)
}

// RestoreSubscriptionResponse represents restore subscription response
type RestoreSubscriptionResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	IsActive   bool   `json:"is_active"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	Plan       string `json:"plan,omitempty"`
	ProductID  string `json:"product_id,omitempty"`
}

// RestoreSubscription restores subscription from latest receipt
// POST /api/subscription/restore
func RestoreSubscription(c *gin.Context) {
	var req RestoreSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RestoreSubscriptionResponse{
			Success: false,
			Message: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Get project by app_id
	projectService := services.NewProjectService()
	project, err := projectService.GetProjectByBundleID(req.AppID)
	if err != nil {
		// Try package name if bundle_id fails
		project, err = projectService.GetProjectByPackageName(req.AppID)
		if err != nil {
			c.JSON(http.StatusBadRequest, RestoreSubscriptionResponse{
				Success: false,
				Message: "App not found: " + err.Error(),
			})
			return
		}
	}

	// Get latest subscription
	subscription, err := database.GetLatestSubscriptionByUser(project.ProjectID, req.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, RestoreSubscriptionResponse{
			Success: false,
			Message: "No subscription found for this user",
		})
		return
	}

	// If there's a latest receipt, re-verify it
	if subscription.LatestReceipt != "" && subscription.Platform == "ios" {
		verificationService := services.NewSubscriptionVerificationService()
		updatedSubscription, err := verificationService.VerifyAppleReceipt(
			project.ProjectID,
			subscription.LatestReceipt,
			req.UserID,
		)
		if err != nil {
			// If verification fails, return existing subscription info
			c.JSON(http.StatusOK, RestoreSubscriptionResponse{
				Success:   true,
				Message:   "Subscription restored (verification failed, using cached data)",
				IsActive:  subscription.Status == "active" && subscription.ExpiresDate.After(time.Now()),
				ExpiresAt: subscription.ExpiresDate.Format(time.RFC3339),
				Plan:      subscription.Plan,
				ProductID: subscription.ProductID,
			})
			return
		}
		subscription = updatedSubscription
	}

	// Check if subscription is active
	isActive := subscription.Status == "active" && subscription.ExpiresDate.After(time.Now())

	c.JSON(http.StatusOK, RestoreSubscriptionResponse{
		Success:   true,
		Message:   "Subscription restored successfully",
		IsActive:  isActive,
		ExpiresAt: subscription.ExpiresDate.Format(time.RFC3339),
		Plan:      subscription.Plan,
		ProductID: subscription.ProductID,
	})
}

