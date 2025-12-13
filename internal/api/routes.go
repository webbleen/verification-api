package api

import (
	"net/http"
	"verification-api/internal/middleware"
	"verification-api/internal/models"
	"verification-api/internal/services"

	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up all routes
func SetupRoutes(r *gin.Engine) {
	// Initialize project manager
	middleware.InitProjectManager()

	// API route group
	api := r.Group("/api")
	{
		// Verification code routes (require project authentication)
		verification := api.Group("/verification")
		verification.Use(middleware.ProjectAuthMiddleware())
		{
			verification.POST("/send-code", SendVerificationCode)
			verification.POST("/verify-code", VerifyCode)
		}

		// Project management routes (for admin use)
		admin := api.Group("/admin")
		{
			admin.GET("/projects", GetProjects)
			admin.POST("/projects", CreateProject)
			admin.PUT("/projects/:id", UpdateProject)
			admin.DELETE("/projects/:id", DeleteProject)
			admin.GET("/projects/:id/stats", GetProjectStats)
		}

		// Statistics and monitoring routes
		stats := api.Group("/stats")
		stats.Use(middleware.ProjectAuthMiddleware())
		{
			stats.GET("/project", GetProjectStats)
		}

		// Subscription routes
		// Note: /status endpoint supports both authenticated (backend) and unauthenticated (client) requests
		subscription := api.Group("/subscription")
		{
			subscription.POST("/verify", VerifySubscription)
			subscription.GET("/status", GetSubscriptionStatus) // Supports both client and backend calls
			subscription.POST("/restore", RestoreSubscription)
			subscription.POST("/bind_account", BindAccount)      // Bind user_id to subscription
			subscription.GET("/history", GetSubscriptionHistory) // Get subscription history
		}

		// Verify routes (for App Server to verify transactions)
		verify := api.Group("/verify")
		verify.Use(middleware.ProjectAuthMiddleware()) // 需要项目认证
		{
			verify.POST("/apple", VerifyApple) // 验证 Apple 交易
		}

		// Webhook routes (no authentication, called by Apple/Google)
		webhook := r.Group("/webhook")
		{
			// Apple webhook routes (separate endpoints for production and sandbox)
			webhook.POST("/apple/production", AppStoreProductionWebhookHandler) // Production environment
			webhook.POST("/apple/sandbox", AppStoreSandboxWebhookHandler)       // Sandbox environment
			webhook.POST("/google", GooglePlayWebhookHandler)                   // Google Play webhook
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "unionhub",
		})
	})
}

// GetProjects gets all projects
func GetProjects(c *gin.Context) {
	projectService := services.NewProjectService()
	projects, err := projectService.GetAllProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get projects",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    projects,
	})
}

// CreateProjectRequest represents create project request
type CreateProjectRequest struct {
	ProjectID          string `json:"project_id" binding:"required"`
	ProjectName        string `json:"project_name" binding:"required"`
	APIKey             string `json:"api_key" binding:"required"`
	FromName           string `json:"from_name" binding:"required"`
	TemplateID         string `json:"template_id"`
	Description        string `json:"description"`
	ContactEmail       string `json:"contact_email"`
	MaxRequests        int    `json:"max_requests"`
	BundleID           string `json:"bundle_id"`            // iOS bundle ID (for subscription center)
	PackageName        string `json:"package_name"`         // Android package name (for subscription center)
	WebhookCallbackURL string `json:"webhook_callback_url"` // App Backend webhook URL (optional)
	WebhookSecret      string `json:"webhook_secret"`       // Webhook signature secret (optional)
}

// CreateProject creates a new project
func CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.MaxRequests == 0 {
		req.MaxRequests = 1000 // 1000 requests per day
	}

	project := &models.Project{
		ProjectID:          req.ProjectID,
		ProjectName:        req.ProjectName,
		APIKey:             req.APIKey,
		FromName:           req.FromName,
		TemplateID:         req.TemplateID,
		Description:        req.Description,
		ContactEmail:       req.ContactEmail,
		MaxRequests:        req.MaxRequests,
		BundleID:           req.BundleID,
		PackageName:        req.PackageName,
		WebhookCallbackURL: req.WebhookCallbackURL,
		WebhookSecret:      req.WebhookSecret,
		IsActive:           true,
	}

	projectService := services.NewProjectService()
	if err := projectService.CreateProject(project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to create project: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Project created successfully",
		"data":    project,
	})
}

// UpdateProjectRequest represents update project request
type UpdateProjectRequest struct {
	ProjectName        string `json:"project_name"`
	FromName           string `json:"from_name"`
	TemplateID         string `json:"template_id"`
	Description        string `json:"description"`
	ContactEmail       string `json:"contact_email"`
	MaxRequests        int    `json:"max_requests"`
	IsActive           *bool  `json:"is_active"`
	BundleID           string `json:"bundle_id"`            // iOS bundle ID
	PackageName        string `json:"package_name"`         // Android package name
	WebhookCallbackURL string `json:"webhook_callback_url"` // App Backend webhook URL (optional)
	WebhookSecret      string `json:"webhook_secret"`       // Webhook signature secret (optional)
}

// UpdateProject updates an existing project
func UpdateProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID is required",
		})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Build update map
	updates := make(map[string]interface{})
	if req.ProjectName != "" {
		updates["project_name"] = req.ProjectName
	}
	if req.FromName != "" {
		updates["from_name"] = req.FromName
	}
	if req.TemplateID != "" {
		updates["template_id"] = req.TemplateID
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.ContactEmail != "" {
		updates["contact_email"] = req.ContactEmail
	}
	if req.MaxRequests > 0 {
		updates["max_requests"] = req.MaxRequests
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.BundleID != "" {
		updates["bundle_id"] = req.BundleID
	}
	if req.PackageName != "" {
		updates["package_name"] = req.PackageName
	}
	// Webhook fields (empty string means remove webhook)
	if req.WebhookCallbackURL != "" || c.Query("remove_webhook") == "true" {
		updates["webhook_callback_url"] = req.WebhookCallbackURL
	}
	if req.WebhookSecret != "" || c.Query("remove_webhook") == "true" {
		updates["webhook_secret"] = req.WebhookSecret
	}

	projectService := services.NewProjectService()
	if err := projectService.UpdateProject(projectID, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to update project: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project updated successfully",
	})
}

// DeleteProject deletes a project
func DeleteProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID is required",
		})
		return
	}

	projectService := services.NewProjectService()
	if err := projectService.DeleteProject(projectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to delete project: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project deleted successfully",
	})
}

// GetProjectStats gets project statistics
func GetProjectStats(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		// If no ID in param, get from context (for stats routes)
		if pid, exists := c.Get("project_id"); exists {
			projectID = pid.(string)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Project ID is required",
			})
			return
		}
	}

	projectService := services.NewProjectService()
	stats, err := projectService.GetProjectStats(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get project stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
