package api

import (
	"net/http"
	"time"
	"verification-api/internal/database"
	"verification-api/internal/models"
	"verification-api/internal/services"

	"github.com/gin-gonic/gin"
)

// GetSubscriptionStatusResponse represents subscription status response
type GetSubscriptionStatusResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message,omitempty"`
	IsActive    bool   `json:"is_active"`
	Platform    string `json:"platform,omitempty"`     // Platform: ios or android
	Status      string `json:"status,omitempty"`        // Subscription status
	Plan        string `json:"plan,omitempty"`
	ExpiresDate string `json:"expires_date,omitempty"` // ISO 8601 format
	ProductID   string `json:"product_id,omitempty"`
	AutoRenew   bool   `json:"auto_renew,omitempty"`

	// Legacy support (deprecated)
	ExpiresAt string `json:"expires_at,omitempty"` // Deprecated: use expires_date
}

// GetSubscriptionStatus gets subscription status
// GET /api/subscription/status?user_id=xxx&app_id=yyy
// Can be called by both client and app backend
func GetSubscriptionStatus(c *gin.Context) {
	userID := c.Query("user_id")
	appID := c.Query("app_id")
	platform := c.DefaultQuery("platform", "ios") // Default to ios

	if userID == "" || appID == "" {
		c.JSON(http.StatusBadRequest, GetSubscriptionStatusResponse{
			Success: false,
			Message: "user_id and app_id are required",
		})
		return
	}

	// Get project by app_id
	projectService := services.NewProjectService()
	var project *models.Project
	var err error

	if platform == "ios" {
		project, err = projectService.GetProjectByBundleID(appID)
	} else {
		project, err = projectService.GetProjectByPackageName(appID)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, GetSubscriptionStatusResponse{
			Success: false,
			Message: "App not found: " + err.Error(),
		})
		return
	}

	// Get active subscription
	subscription, err := database.GetActiveSubscription(project.ProjectID, userID)
	if err != nil {
		// No active subscription found
		c.JSON(http.StatusOK, GetSubscriptionStatusResponse{
			Success: true,
			IsActive: false,
			Status:   "inactive",
		})
		return
	}

	// Check if subscription is still active
	isActive := subscription.Status == "active" && subscription.ExpiresDate.After(time.Now())

	c.JSON(http.StatusOK, GetSubscriptionStatusResponse{
		Success:     true,
		IsActive:    isActive,
		Platform:    subscription.Platform,
		Status:      subscription.Status,
		Plan:        subscription.Plan,
		ExpiresDate: subscription.ExpiresDate.Format(time.RFC3339),
		ExpiresAt:   subscription.ExpiresDate.Format(time.RFC3339), // Legacy support
		ProductID:   subscription.ProductID,
		AutoRenew:   subscription.AutoRenewStatus,
	})
}

