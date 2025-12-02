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

		// Subscription routes (client API - no authentication required)
		subscription := api.Group("/subscription")
		{
			subscription.POST("/verify", VerifySubscription)
			subscription.GET("/status", GetSubscriptionStatus)
			subscription.POST("/restore", RestoreSubscription)
		}

		// Subscription routes for app backend (requires project authentication)
		subscriptionBackend := api.Group("/subscription")
		subscriptionBackend.Use(middleware.ProjectAuthMiddleware())
		{
			subscriptionBackend.GET("/status", GetSubscriptionStatus) // Same endpoint, but with auth
		}

		// App Store notification routes (no authentication, Apple calls these)
		appstore := api.Group("/appstore")
		{
			appstore.POST("/notifications/production", AppStoreProductionNotificationHandler)
			appstore.POST("/notifications/sandbox", AppStoreSandboxNotificationHandler)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "verification-service",
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
	ProjectID    string `json:"project_id" binding:"required"`
	ProjectName  string `json:"project_name" binding:"required"`
	APIKey       string `json:"api_key" binding:"required"`
	FromName     string `json:"from_name" binding:"required"`
	TemplateID   string `json:"template_id"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	RateLimit    int    `json:"rate_limit"`
	MaxRequests  int    `json:"max_requests"`
	BundleID     string `json:"bundle_id"`    // iOS bundle ID (for subscription center)
	PackageName  string `json:"package_name"` // Android package name (for subscription center)
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
	if req.RateLimit == 0 {
		req.RateLimit = 60 // 60 requests per hour
	}
	if req.MaxRequests == 0 {
		req.MaxRequests = 1000 // 1000 requests per day
	}

	project := &models.Project{
		ProjectID:    req.ProjectID,
		ProjectName:  req.ProjectName,
		APIKey:       req.APIKey,
		FromName:     req.FromName,
		TemplateID:   req.TemplateID,
		Description:  req.Description,
		ContactEmail: req.ContactEmail,
		RateLimit:    req.RateLimit,
		MaxRequests:  req.MaxRequests,
		BundleID:     req.BundleID,
		PackageName:  req.PackageName,
		IsActive:     true,
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
	ProjectName  string `json:"project_name"`
	FromName     string `json:"from_name"`
	TemplateID   string `json:"template_id"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	RateLimit    int    `json:"rate_limit"`
	MaxRequests  int    `json:"max_requests"`
	IsActive     *bool  `json:"is_active"`
	BundleID     string `json:"bundle_id"`    // iOS bundle ID
	PackageName  string `json:"package_name"` // Android package name
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
	if req.RateLimit > 0 {
		updates["rate_limit"] = req.RateLimit
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
