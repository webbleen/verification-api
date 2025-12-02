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
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	IsActive   bool   `json:"is_active"`
	Status     string `json:"status,omitempty"`
	Plan       string `json:"plan,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
	ProductID  string `json:"product_id,omitempty"`
	AutoRenew  bool   `json:"auto_renew,omitempty"`
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
		Success:   true,
		IsActive:  isActive,
		Status:    subscription.Status,
		Plan:      subscription.Plan,
		ExpiresAt: subscription.ExpiresDate.Format(time.RFC3339),
		ProductID: subscription.ProductID,
		AutoRenew: subscription.AutoRenewStatus,
	})
}

